/*
Copyright 2024 The Tekton Authors

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

package taskrunmetrics

import (
	"context"
	"sync"
	"testing"

	"knative.dev/pkg/metrics/metricstest"
	_ "knative.dev/pkg/metrics/testing"

	"github.com/tektoncd/chains/pkg/metrics"
)

func TestUninitializedMetrics(t *testing.T) {
	recorder := &Recorder{}
	ctx := context.Background()

	recorder.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	metricstest.CheckStatsNotReported(t, string(taskRunSignedName))

	recorder.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	metricstest.CheckStatsNotReported(t, string(taskRunUploadedName))

	recorder.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	metricstest.CheckStatsNotReported(t, string(taskRunStoredName))

	recorder.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
	metricstest.CheckStatsNotReported(t, string(taskRunMarkedName))
}

func TestCountMetrics(t *testing.T) {
	unregisterMetrics()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)

	rec.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	metricstest.CheckCountData(t, string(taskRunSignedName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	metricstest.CheckCountData(t, string(taskRunUploadedName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	metricstest.CheckCountData(t, string(taskRunStoredName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
	metricstest.CheckCountData(t, string(taskRunMarkedName), map[string]string{}, 1)
}

func TestRecordErrorMetric(t *testing.T) {
	unregisterMetrics()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)
	if rec == nil {
		t.Fatal("Recorder not initialized")
	}

	// Record an error metric with a sample error type "signing"
	rec.RecordErrorMetric(ctx, "signing")

	// Verify that the error metric is recorded with the tag error_type=signing.
	metricstest.CheckCountData(t, string(taskRunErrorCountName), map[string]string{"error_type": "signing"}, 1)
}

func unregisterMetrics() {
	metricstest.Unregister(
		string(taskRunSignedName),
		string(taskRunUploadedName),
		string(taskRunStoredName),
		string(taskRunMarkedName),
		string(taskRunErrorCountName),
	)
	// Allow the recorder singleton to be recreated.
	once = sync.Once{}
	r = nil
}
