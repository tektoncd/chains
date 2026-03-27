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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"knative.dev/pkg/logging"

	common "github.com/tektoncd/chains/pkg/metrics"
)

const (
	taskRunSignedName     common.Metric = "watcher_taskrun_sign_created_total"
	taskRunSignedDesc     string        = "Total number of signed messages for taskruns"
	taskRunUploadedName   common.Metric = "watcher_taskrun_payload_uploaded_total"
	taskRunUploadedDesc   string        = "Total number of uploaded payloads for taskruns"
	taskRunStoredName     common.Metric = "watcher_taskrun_payload_stored_total"
	taskRunStoredDesc     string        = "Total number of stored payloads for taskruns"
	taskRunMarkedName     common.Metric = "watcher_taskrun_marked_signed_total"
	taskRunMarkedDesc     string        = "Total number of objects marked as signed for taskruns"
	taskRunErrorCountName common.Metric = "watcher_taskrun_signing_failures_total"
	taskRunErrorCountDesc string        = "Total number of TaskRun signing failures"
)

var _ common.Recorder = &Recorder{}

// Recorder is used to actually record TaskRun metrics.
type Recorder struct {
	initialized bool
	sgCount     otelmetric.Int64Counter
	plCount     otelmetric.Int64Counter
	stCount     otelmetric.Int64Counter
	mrCount     otelmetric.Int64Counter
	errCount    otelmetric.Int64Counter
}

// NewRecorder lazily initializes this singleton. Unlike sync.Once, the
// mutex-based guard allows a retry if initialization fails (e.g. the OTel
// provider is not yet ready). Only a fully-initialized recorder is stored in r.
var (
	mu sync.Mutex
	r  *Recorder
)

// NewRecorder creates a new metrics recorder instance
// to log the TaskRun related metrics.
func NewRecorder(ctx context.Context) (*Recorder, error) {
	mu.Lock()
	defer mu.Unlock()
	if r != nil && r.initialized {
		return r, nil
	}

	logger := logging.FromContext(ctx)
	newR := &Recorder{}
	meter := otel.Meter("github.com/tektoncd/chains/pkg/taskrunmetrics")

	var err error
	newR.sgCount, err = meter.Int64Counter(
		string(taskRunSignedName),
		otelmetric.WithDescription(taskRunSignedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", taskRunSignedName, err)
		return nil, err
	}

	newR.plCount, err = meter.Int64Counter(
		string(taskRunUploadedName),
		otelmetric.WithDescription(taskRunUploadedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", taskRunUploadedName, err)
		return nil, err
	}

	newR.stCount, err = meter.Int64Counter(
		string(taskRunStoredName),
		otelmetric.WithDescription(taskRunStoredDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", taskRunStoredName, err)
		return nil, err
	}

	newR.mrCount, err = meter.Int64Counter(
		string(taskRunMarkedName),
		otelmetric.WithDescription(taskRunMarkedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", taskRunMarkedName, err)
		return nil, err
	}

	newR.errCount, err = meter.Int64Counter(
		string(taskRunErrorCountName),
		otelmetric.WithDescription(taskRunErrorCountDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", taskRunErrorCountName, err)
		return nil, err
	}

	newR.initialized = true
	r = newR
	return r, nil
}

// RecordCountMetrics implements github.com/tektoncd/chains/pkg/metrics.Recorder.RecordCountMetrics
func (r *Recorder) RecordCountMetrics(ctx context.Context, metricType common.Metric) {
	if r == nil {
		return
	}
	logger := logging.FromContext(ctx)
	if !r.initialized {
		logger.Debugf("Ignoring the metrics recording as recorder not initialized")
		return
	}
	switch mt := metricType; mt {
	case common.SignedMessagesCount:
		r.sgCount.Add(ctx, 1)
	case common.PayloadUploadedCount:
		r.plCount.Add(ctx, 1)
	case common.SignsStoredCount:
		r.stCount.Add(ctx, 1)
	case common.MarkedAsSignedCount:
		r.mrCount.Add(ctx, 1)
	default:
		logger.Errorf("Ignoring the metrics recording as valid Metric type matching %v was not found", mt)
	}
}

// RecordErrorMetric records a TaskRun signing failure with a given error type.
func (r *Recorder) RecordErrorMetric(ctx context.Context, errType common.MetricErrorType) {
	if r == nil {
		return
	}
	if !r.initialized {
		return
	}
	r.errCount.Add(ctx, 1, otelmetric.WithAttributes(attribute.String(common.ErrorTypeAttrKey, string(errType))))
}
