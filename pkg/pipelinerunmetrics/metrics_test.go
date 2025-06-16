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

package pipelinerunmetrics

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
	metricstest.CheckStatsNotReported(t, string(pipelineRunSignedName))

	recorder.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	metricstest.CheckStatsNotReported(t, string(pipelineRunUploadedName))

	recorder.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	metricstest.CheckStatsNotReported(t, string(pipelineRunStoredName))

	recorder.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
	metricstest.CheckStatsNotReported(t, string(pipelineRunMarkedName))
}

func TestCountMetrics(t *testing.T) {
	unregisterMetrics()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)

	rec.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	metricstest.CheckCountData(t, string(pipelineRunSignedName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	metricstest.CheckCountData(t, string(pipelineRunUploadedName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	metricstest.CheckCountData(t, string(pipelineRunStoredName), map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
	metricstest.CheckCountData(t, string(pipelineRunMarkedName), map[string]string{}, 1)
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
	metricstest.CheckCountData(t, string(pipelineRunErrorCountName), map[string]string{"error_type": "signing"}, 1)
}

func unregisterMetrics() {
	metricstest.Unregister(
		string(pipelineRunSignedName),
		string(pipelineRunUploadedName),
		string(pipelineRunStoredName),
		string(pipelineRunMarkedName),
		string(pipelineRunErrorCountName),
	)
	// Allow the recorder singleton to be recreated.
	once = sync.Once{}
	r = nil
}
