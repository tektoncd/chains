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
	pipelineRunSignedName     common.Metric = "pipelinerun_sign_created_total"
	pipelineRunSignedDesc     string        = "Total number of signed messages for pipelineruns"
	pipelineRunUploadedName   common.Metric = "pipelinerun_payload_uploaded_total"
	pipelineRunUploadedDesc   string        = "Total number of uploaded payloads for pipelineruns"
	pipelineRunStoredName     common.Metric = "pipelinerun_payload_stored_total"
	pipelineRunStoredDesc     string        = "Total number of stored payloads for pipelineruns"
	pipelineRunMarkedName     common.Metric = "pipelinerun_marked_signed_total"
	pipelineRunMarkedDesc     string        = "Total number of objects marked as signed for pipelineruns"
	pipelineRunErrorCountName common.Metric = "pipelinerun_signing_failures_total"
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

// We cannot register the view multiple times, so NewRecorder lazily
// initializes this singleton and returns the same recorder across any
// subsequent invocations.
var (
	once sync.Once
	r    *Recorder
)

// NewRecorder creates a new metrics recorder instance
// to log the PipelineRun related metrics.
func NewRecorder(ctx context.Context) (*Recorder, error) {
	var errRegistering error
	logger := logging.FromContext(ctx)
	once.Do(func() {
		r = &Recorder{}
		meter := otel.Meter("pipelinerunmetrics")

		r.sgCount, errRegistering = meter.Int64Counter(
			string(pipelineRunSignedName),
			otelmetric.WithDescription(pipelineRunSignedDesc),
		)
		if errRegistering != nil {
			logger.Errorf("Failed to create %s counter: %v", pipelineRunSignedName, errRegistering)
			return
		}

		r.plCount, errRegistering = meter.Int64Counter(
			string(pipelineRunUploadedName),
			otelmetric.WithDescription(pipelineRunUploadedDesc),
		)
		if errRegistering != nil {
			logger.Errorf("Failed to create %s counter: %v", pipelineRunUploadedName, errRegistering)
			return
		}

		r.stCount, errRegistering = meter.Int64Counter(
			string(pipelineRunStoredName),
			otelmetric.WithDescription(pipelineRunStoredDesc),
		)
		if errRegistering != nil {
			logger.Errorf("Failed to create %s counter: %v", pipelineRunStoredName, errRegistering)
			return
		}

		r.mrCount, errRegistering = meter.Int64Counter(
			string(pipelineRunMarkedName),
			otelmetric.WithDescription(pipelineRunMarkedDesc),
		)
		if errRegistering != nil {
			logger.Errorf("Failed to create %s counter: %v", pipelineRunMarkedName, errRegistering)
			return
		}

		r.errCount, errRegistering = meter.Int64Counter(
			string(pipelineRunErrorCountName),
			otelmetric.WithDescription(pipelineRunErrorCountDesc),
		)
		if errRegistering != nil {
			logger.Errorf("Failed to create %s counter: %v", pipelineRunErrorCountName, errRegistering)
			return
		}

		r.initialized = true
	})

	return r, errRegistering
}

// RecordCountMetrics implements github.com/tektoncd/chains/pkg/metrics.Recorder.RecordCountMetrics
func (r *Recorder) RecordCountMetrics(ctx context.Context, metricType common.Metric) {
	logger := logging.FromContext(ctx)
	if !r.initialized {
		logger.Errorf("Ignoring the metrics recording as recorder not initialized ")
		return
	}
	switch mt := metricType; mt {
	case common.SignedMessagesCount:
		r.sgCount.Add(ctx, 1)
	case common.PayloadUploadeCount:
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
	if !r.initialized {
		return
	}
	r.errCount.Add(ctx, 1, otelmetric.WithAttributes(attribute.String("error_type", string(errType))))
}
