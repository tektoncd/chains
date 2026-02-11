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

	"github.com/tektoncd/chains/pkg/metrics"
)

func TestUninitializedMetrics(t *testing.T) {
	recorder := &Recorder{}
	ctx := context.Background()

	// These should not panic even though recorder is not initialized
	recorder.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	recorder.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	recorder.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	recorder.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
}

func TestCountMetrics(t *testing.T) {
	resetRecorder()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)
	if rec == nil {
		t.Fatal("Failed to initialize recorder")
	}

	// Record metrics - these should not panic
	rec.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	rec.RecordCountMetrics(ctx, metrics.PayloadUploadeCount)
	rec.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	rec.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
}

func TestRecordErrorMetric(t *testing.T) {
	resetRecorder()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)
	if rec == nil {
		t.Fatal("Recorder not initialized")
	}

	// Record an error metric with a sample error type "signing"
	rec.RecordErrorMetric(ctx, "signing")
}

func resetRecorder() {
	// Allow the recorder singleton to be recreated.
	once = sync.Once{}
	r = nil
}
