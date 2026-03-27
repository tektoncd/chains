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
	"testing"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/tektoncd/chains/pkg/metrics"
)

func resetRecorder() {
	r = nil
}

func setupTestMeterProvider(t *testing.T) (*sdkmetric.ManualReader, func()) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prevProvider := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	return reader, func() {
		otel.SetMeterProvider(prevProvider)
	}
}

func TestUninitializedMetrics(t *testing.T) {
	reader, cleanup := setupTestMeterProvider(t)
	defer cleanup()

	recorder := &Recorder{}
	ctx := context.Background()

	// Should not panic or crash when recorder is uninitialized
	recorder.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	recorder.RecordCountMetrics(ctx, metrics.PayloadUploadedCount)
	recorder.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	recorder.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)
	recorder.RecordErrorMetric(ctx, metrics.SigningError)

	// No metrics should have been recorded
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	for _, sm := range rm.ScopeMetrics {
		if len(sm.Metrics) > 0 {
			t.Errorf("expected no metrics from uninitialized recorder, got %d metric(s) in scope %q", len(sm.Metrics), sm.Scope.Name)
		}
	}
}

func TestCountMetrics(t *testing.T) {
	resetRecorder()
	reader, cleanup := setupTestMeterProvider(t)
	defer cleanup()

	ctx := context.Background()
	ctx = WithClient(ctx)
	rec := Get(ctx)

	rec.RecordCountMetrics(ctx, metrics.SignedMessagesCount)
	rec.RecordCountMetrics(ctx, metrics.PayloadUploadedCount)
	rec.RecordCountMetrics(ctx, metrics.SignsStoredCount)
	rec.RecordCountMetrics(ctx, metrics.MarkedAsSignedCount)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	checkCounterValue(t, rm, string(taskRunSignedName))
	checkCounterValue(t, rm, string(taskRunUploadedName))
	checkCounterValue(t, rm, string(taskRunStoredName))
	checkCounterValue(t, rm, string(taskRunMarkedName))
}

func TestRecordErrorMetric(t *testing.T) {
	resetRecorder()
	reader, cleanup := setupTestMeterProvider(t)
	defer cleanup()

	ctx := context.Background()
	ctx = WithClient(ctx)
	rec := Get(ctx)
	if rec == nil {
		t.Fatal("Recorder not initialized")
	}

	rec.RecordErrorMetric(ctx, metrics.SigningError)

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}

	checkErrorCounterValue(t, rm, string(taskRunErrorCountName), string(metrics.SigningError), 1)
}

func checkCounterValue(t *testing.T, rm metricdata.ResourceMetrics, name string) {
	t.Helper()
	const expected int64 = 1
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				sum, ok := m.Data.(metricdata.Sum[int64])
				if !ok {
					t.Errorf("metric %q has unexpected data type: %T", name, m.Data)
					return
				}
				if len(sum.DataPoints) == 0 {
					t.Errorf("metric %q has no data points", name)
					return
				}
				if sum.DataPoints[0].Value != expected {
					t.Errorf("metric %q: got %d, want %d", name, sum.DataPoints[0].Value, expected)
				}
				return
			}
		}
	}
	t.Errorf("metric %q not found in collected metrics", name)
}

func checkErrorCounterValue(t *testing.T, rm metricdata.ResourceMetrics, name, errType string, expected int64) {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Errorf("metric %q has unexpected data type: %T", name, m.Data)
				return
			}
			for _, dp := range sum.DataPoints {
				for _, attr := range dp.Attributes.ToSlice() {
					if string(attr.Key) == metrics.ErrorTypeAttrKey && attr.Value.AsString() == errType {
						if dp.Value != expected {
							t.Errorf("metric %q with error_type=%q: got %d, want %d", name, errType, dp.Value, expected)
						}
						return
					}
				}
			}
			t.Errorf("metric %q with error_type=%q not found in data points", name, errType)
			return
		}
	}
	t.Errorf("metric %q not found in collected metrics", name)
}
