//go:build e2e
// +build e2e

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
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/test/tekton"
	"k8s.io/apimachinery/pkg/util/wait"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	metricsServiceName = "tekton-chains-metrics"
	metricsPort        = 9090
	metricsPath        = "/metrics"
)

// TestMetrics verifies that the Chains metrics endpoint is reachable and that
// at least one watcher_taskrun_* counter is recorded after a TaskRun is signed.
func TestMetrics(t *testing.T) {
	ctx := logtesting.TestContextWithLogger(t)
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	t.Cleanup(cleanup)

	resetConfig := setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.format":  "in-toto",
		"artifacts.taskrun.signer":  "x509",
		"artifacts.taskrun.storage": "tekton",
	})
	t.Cleanup(resetConfig)

	trObj := tekton.CreateObject(t, ctx, c.PipelineClient, getTaskRunObject(ns))
	if o := waitForCondition(ctx, t, c.PipelineClient, trObj, done, time.Minute); o == nil {
		t.Fatal("TaskRun never completed.")
	}
	if o := waitForCondition(ctx, t, c.PipelineClient, trObj, signed, 2*time.Minute); o == nil {
		t.Fatal("TaskRun never signed by Chains.")
	}

	assertMetricsPresent(ctx, t, []string{
		"watcher_taskrun_sign_created_total",
	})
}

// assertMetricsPresent scrapes the Prometheus /metrics endpoint via
// kubectl port-forward and fails if any of the given metric sample lines are absent.
func assertMetricsPresent(ctx context.Context, t *testing.T, metricNames []string) {
	t.Helper()

	localPort, stopFwd := kubectlPortForward(ctx, t, namespace, metricsServiceName, metricsPort)
	defer stopFwd()

	url := fmt.Sprintf("http://localhost:%d%s", localPort, metricsPath)

	// Retry for up to 30 s in case the Prometheus exporter has not flushed yet.
	var body string
	if err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 30*time.Second, true, func(context.Context) (bool, error) {
		resp, err := http.Get(url) //nolint:noctx
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false, nil
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		body = string(b)
		return true, nil
	}); err != nil {
		t.Fatalf("Metrics endpoint %s never returned HTTP 200: %v", url, err)
	}

	missing := findMissingMetrics(body, metricNames)
	if len(missing) > 0 {
		t.Errorf("The following metrics were not found in %s:\n  %s\n\nFull output:\n%s",
			url, strings.Join(missing, "\n  "), body)
	}
}

// findMissingMetrics returns entries from want whose name does not appear as a
// sample line (not a HELP/TYPE comment) in the Prometheus text-format body.
func findMissingMetrics(body string, want []string) []string {
	present := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		for _, name := range want {
			if strings.HasPrefix(line, name+"{") || strings.HasPrefix(line, name+" ") {
				present[name] = true
			}
		}
	}

	var missing []string
	for _, name := range want {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// kubectlPortForward starts a background kubectl port-forward tunnel from a
// free local port to remotePort on svc/svcName in the given namespace.
// It blocks until the tunnel is accepting connections (up to 30 s) and returns
// the local port plus a stop function that kills the background process.
func kubectlPortForward(ctx context.Context, t *testing.T, ns, svcName string, remotePort int) (int, func()) {
	t.Helper()

	// Find a free local port.
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to find free port for port-forward: %v", err)
	}
	localPort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	target := fmt.Sprintf("svc/%s", svcName)
	mapping := fmt.Sprintf("%d:%d", localPort, remotePort)

	// #nosec G204 -- ns, svcName and mapping are controlled, not user-supplied
	cmd := exec.CommandContext(ctx, "kubectl", "port-forward",
		"--namespace", ns,
		target, mapping,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start kubectl port-forward: %v", err)
	}
	t.Logf("kubectl port-forward %s %s started (pid %d)", target, mapping, cmd.Process.Pid)

	stop := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}

	// Poll until the local port accepts TCP connections (up to 30 s).
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		dialCtx, dialCancel := context.WithTimeout(ctx, time.Second)
		conn, dialErr := (&net.Dialer{}).DialContext(dialCtx, "tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
		dialCancel()
		if dialErr == nil {
			conn.Close()
			t.Logf("Port-forward ready on localhost:%d", localPort)
			return localPort, stop
		}
		time.Sleep(500 * time.Millisecond)
	}

	stop()
	t.Fatalf("Timed out waiting for kubectl port-forward on localhost:%d", localPort)
	return 0, stop
}
