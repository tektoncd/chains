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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"knative.dev/pkg/logging"

	common "github.com/tektoncd/chains/pkg/metrics"
)

const (
	pipelineRunSignedName     common.Metric = "watcher_pipelinerun_sign_created_total"
	pipelineRunSignedDesc     string        = "Total number of signed messages for pipelineruns"
	pipelineRunUploadedName   common.Metric = "watcher_pipelinerun_payload_uploaded_total"
	pipelineRunUploadedDesc   string        = "Total number of uploaded payloads for pipelineruns"
	pipelineRunStoredName     common.Metric = "watcher_pipelinerun_payload_stored_total"
	pipelineRunStoredDesc     string        = "Total number of stored payloads for pipelineruns"
	pipelineRunMarkedName     common.Metric = "watcher_pipelinerun_marked_signed_total"
	pipelineRunMarkedDesc     string        = "Total number of objects marked as signed for pipelineruns"
	pipelineRunErrorCountName common.Metric = "watcher_pipelinerun_signing_failures_total"
	pipelineRunErrorCountDesc string        = "Total number of PipelineRun signing failures"
)

var _ common.Recorder = &Recorder{}

// Recorder holds keys for PipelineRun metrics.
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
// to log PipelineRun related metrics.
func NewRecorder(ctx context.Context) (*Recorder, error) {
	mu.Lock()
	defer mu.Unlock()
	if r != nil && r.initialized {
		return r, nil
	}

	logger := logging.FromContext(ctx)
	newR := &Recorder{}
	meter := otel.Meter("github.com/tektoncd/chains/pkg/pipelinerunmetrics")

	var err error
	newR.sgCount, err = meter.Int64Counter(
		string(pipelineRunSignedName),
		otelmetric.WithDescription(pipelineRunSignedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", pipelineRunSignedName, err)
		return nil, err
	}

	newR.plCount, err = meter.Int64Counter(
		string(pipelineRunUploadedName),
		otelmetric.WithDescription(pipelineRunUploadedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", pipelineRunUploadedName, err)
		return nil, err
	}

	newR.stCount, err = meter.Int64Counter(
		string(pipelineRunStoredName),
		otelmetric.WithDescription(pipelineRunStoredDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", pipelineRunStoredName, err)
		return nil, err
	}

	newR.mrCount, err = meter.Int64Counter(
		string(pipelineRunMarkedName),
		otelmetric.WithDescription(pipelineRunMarkedDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", pipelineRunMarkedName, err)
		return nil, err
	}

	newR.errCount, err = meter.Int64Counter(
		string(pipelineRunErrorCountName),
		otelmetric.WithDescription(pipelineRunErrorCountDesc),
	)
	if err != nil {
		logger.Errorf("Failed to create %s counter: %v", pipelineRunErrorCountName, err)
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

// RecordErrorMetric records a PipelineRun signing failure with a given error type.
func (r *Recorder) RecordErrorMetric(ctx context.Context, errType common.MetricErrorType) {
	if r == nil {
		return
	}
	if !r.initialized {
		return
	}
	r.errCount.Add(ctx, 1, otelmetric.WithAttributes(attribute.String(common.ErrorTypeAttrKey, string(errType))))
}
