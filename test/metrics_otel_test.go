//go:build e2e

/*
Copyright 2026 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/tektoncd/chains/pkg/test/tekton"
	"k8s.io/apimachinery/pkg/util/wait"
	logtesting "knative.dev/pkg/logging/testing"
)

// scrapeMetrics port-forwards to the chains metrics service and returns all
// Prometheus metric families parsed from the /metrics endpoint.
func scrapeMetrics(ctx context.Context, t *testing.T) map[string]*dto.MetricFamily {
	t.Helper()
	localPort, stopFwd := kubectlPortForward(ctx, t, namespace, metricsServiceName, metricsPort)
	defer stopFwd()

	url := fmt.Sprintf("http://localhost:%d%s", localPort, metricsPath)

	var families map[string]*dto.MetricFamily
	if err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 30*time.Second, true, func(context.Context) (bool, error) {
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		parser := expfmt.NewTextParser(model.LegacyValidation)
		parsed, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
		if err != nil {
			return false, nil
		}
		families = parsed
		return true, nil
	}); err != nil {
		t.Fatalf("Metrics endpoint %s never returned parseable output: %v", url, err)
	}
	return families
}

// counterValue returns the sum of counter values for the given metric name.
func counterValue(families map[string]*dto.MetricFamily, name string) float64 {
	fam, ok := families[name]
	if !ok {
		return 0
	}
	var total float64
	for _, m := range fam.GetMetric() {
		if c := m.GetCounter(); c != nil {
			total += c.GetValue()
		}
	}
	return total
}

// counterDelta returns the increase in a counter between two scrapes, guarding
// against resets (returns 0 if the counter decreased).
func counterDeltaChains(after, before map[string]*dto.MetricFamily, name string) float64 {
	a := counterValue(after, name)
	b := counterValue(before, name)
	if a < b {
		return 0
	}
	return a - b
}

// waitForMetricFamily polls until the named metric family appears.
func waitForMetricFamily(ctx context.Context, t *testing.T, name string, timeout time.Duration) map[string]*dto.MetricFamily {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		families := scrapeMetrics(ctx, t)
		if _, ok := families[name]; ok {
			return families
		}
		select {
		case <-ctx.Done():
			t.Fatalf("Timed out waiting for metric %q to appear: %v", name, ctx.Err())
			return nil
		case <-time.After(5 * time.Second):
		}
	}
}

// TestOTelMetrics is a consolidated e2e test for the OpenCensus-to-OpenTelemetry
// metrics migration in Chains (PR #1550). It creates a TaskRun and a PipelineRun,
// waits for Chains to sign them, then scrapes the controller /metrics endpoint to
// verify all counter metrics are present and have incremented.
//
// Metrics verified:
//   - watcher_taskrun_sign_created_total
//   - watcher_taskrun_payload_uploaded_total
//   - watcher_taskrun_payload_stored_total
//   - watcher_taskrun_marked_signed_total
//   - watcher_pipelinerun_sign_created_total
//   - watcher_pipelinerun_payload_uploaded_total
//   - watcher_pipelinerun_payload_stored_total
//   - watcher_pipelinerun_marked_signed_total
func TestOTelMetrics(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.format":      "in-toto",
		"artifacts.taskrun.signer":      "x509",
		"artifacts.taskrun.storage":     "tekton",
		"artifacts.pipelinerun.format":  "in-toto",
		"artifacts.pipelinerun.signer":  "x509",
		"artifacts.pipelinerun.storage": "tekton",
	})
	t.Cleanup(resetConfig)

	// Baseline scrape before creating any resources so assertions can use deltas.
	baseline := scrapeMetrics(ctx, t)

	// ========== Create and sign a TaskRun ==========

	trObj := tekton.CreateObject(t, ctx, c.PipelineClient, getTaskRunObject(ns))
	if o := waitForCondition(ctx, t, c.PipelineClient, trObj, done, time.Minute); o == nil {
		t.Fatal("TaskRun never completed.")
	}
	if o := waitForCondition(ctx, t, c.PipelineClient, trObj, signed, 2*time.Minute); o == nil {
		t.Fatal("TaskRun never signed by Chains.")
	}

	// ========== Create and sign a PipelineRun ==========

	prObj := tekton.CreateObject(t, ctx, c.PipelineClient, getPipelineRunObject(ns))
	if o := waitForCondition(ctx, t, c.PipelineClient, prObj, done, time.Minute); o == nil {
		t.Fatal("PipelineRun never completed.")
	}
	if o := waitForCondition(ctx, t, c.PipelineClient, prObj, signed, 2*time.Minute); o == nil {
		t.Fatal("PipelineRun never signed by Chains.")
	}

	// ========== Scrape metrics and wait for sign counter to appear ==========

	t.Log("Waiting for watcher_taskrun_sign_created_total to appear")
	families := waitForMetricFamily(ctx, t, "watcher_taskrun_sign_created_total", 2*time.Minute)
	t.Logf("Scraped %d metric families", len(families))

	// ========== Counter delta assertions (TaskRun + PipelineRun) ==========

	// watcher_taskrun_payload_uploaded_total and
	// watcher_pipelinerun_payload_uploaded_total only fire for external storage
	// backends (OCI, GCS, etc.). With storage=tekton the payload is written
	// directly to the TaskRun/PipelineRun annotation, so those counters stay
	// at 0 and are not included in the table below.
	tests := []struct {
		name       string
		metricName string
		wantMin    float64
	}{
		{"TaskRun/sign_created_total", "watcher_taskrun_sign_created_total", 1},
		{"TaskRun/payload_stored_total", "watcher_taskrun_payload_stored_total", 1},
		{"TaskRun/marked_signed_total", "watcher_taskrun_marked_signed_total", 1},
		{"PipelineRun/sign_created_total", "watcher_pipelinerun_sign_created_total", 1},
		{"PipelineRun/payload_stored_total", "watcher_pipelinerun_payload_stored_total", 1},
		{"PipelineRun/marked_signed_total", "watcher_pipelinerun_marked_signed_total", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := counterDeltaChains(families, baseline, tt.metricName)
			if delta < tt.wantMin {
				t.Errorf("%s delta = %v, want >= %v", tt.metricName, delta, tt.wantMin)
			}
			t.Logf("%s delta: %v", tt.metricName, delta)
		})
	}

	// ========== Knative infrastructure metrics (OTel renames) ==========

	prefixTests := []struct {
		name   string
		prefix string
		errMsg string
	}{
		{"Renames/workqueue_uses_kn_prefix", "kn_workqueue_", "Expected at least one kn_workqueue_* metric, found none"},
		{"Renames/go_runtime_uses_standard_prefix", "go_", "Expected standard go_* runtime metrics, found none"},
	}
	for _, tt := range prefixTests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for name := range families {
				if strings.HasPrefix(name, tt.prefix) {
					found = true
					break
				}
			}
			if !found {
				t.Error(tt.errMsg)
			}
		})
	}

	// ========== Removed OpenCensus metrics ==========
	// These metrics were present in the OpenCensus-based implementation and
	// must be absent after the OTel migration.
	// TODO: Remove these assertions in a future release once no OC-based
	// release is supported.

	removedMetrics := []string{
		"tekton_chains_taskrun_signed_total",
		"tekton_chains_pipelinerun_signed_total",
	}
	for _, name := range removedMetrics {
		if _, ok := families[name]; ok {
			t.Errorf("Old OC metric %s still present; expected removal after OTel migration", name)
		}
	}
}
