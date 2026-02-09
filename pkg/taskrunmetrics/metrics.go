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
	"go.opentelemetry.io/otel/metric"
	"knative.dev/pkg/logging"

	common "github.com/tektoncd/chains/pkg/metrics"
)

const (
	taskRunSignedName     common.Metric = "taskrun_sign_created_total"
	taskRunSignedDesc     string        = "Total number of signed messages for taskruns"
	taskRunUploadedName   common.Metric = "taskrun_payload_uploaded_total"
	taskRunUploadedDesc   string        = "Total number of uploaded payloads for taskruns"
	taskRunStoredName     common.Metric = "taskrun_payload_stored_total"
	taskRunStoredDesc     string        = "Total number of stored payloads for taskruns"
	taskRunMarkedName     common.Metric = "taskrun_marked_signed_total"
	taskRunMarkedDesc     string        = "Total number of objects marked as signed for taskruns"
	taskRunErrorCountName common.Metric = "taskrun_signing_failures_total"
	taskRunErrorCountDesc string        = "Total number of TaskRun signing failures"
)

var _ common.Recorder = &Recorder{}

// Recorder is used to actually record TaskRun metrics.
type Recorder struct {
	initialized     bool
	meter           metric.Meter
	mutex           sync.Mutex
	signedCounter   metric.Int64Counter
	uploadedCounter metric.Int64Counter
	storedCounter   metric.Int64Counter
	markedCounter   metric.Int64Counter
	errorCounter    metric.Int64Counter
}

// We cannot register the view multiple times, so NewRecorder lazily
// initializes this singleton and returns the same recorder across any
// subsequent invocations.
var (
	once sync.Once
	r    *Recorder
)

// NewRecorder creates a new metrics recorder instance
// to log the TaskRun related metrics.
func NewRecorder(ctx context.Context) (*Recorder, error) {
	var errRegistering error
	logger := logging.FromContext(ctx)
	once.Do(func() {
		r = &Recorder{
			initialized: true,
			meter:       otel.GetMeterProvider().Meter("tekton_chains"),
		}
		errRegistering = r.registerMetrics()
		if errRegistering != nil {
			r.initialized = false
			logger.Errorf("Failed to register metrics: %v", errRegistering)
			return
		}
	})

	return r, errRegistering
}

func (r *Recorder) registerMetrics() error {
	var err error

	r.signedCounter, err = r.meter.Int64Counter(
		string(taskRunSignedName),
		metric.WithDescription(taskRunSignedDesc),
	)
	if err != nil {
		return err
	}

	r.uploadedCounter, err = r.meter.Int64Counter(
		string(taskRunUploadedName),
		metric.WithDescription(taskRunUploadedDesc),
	)
	if err != nil {
		return err
	}

	r.storedCounter, err = r.meter.Int64Counter(
		string(taskRunStoredName),
		metric.WithDescription(taskRunStoredDesc),
	)
	if err != nil {
		return err
	}

	r.markedCounter, err = r.meter.Int64Counter(
		string(taskRunMarkedName),
		metric.WithDescription(taskRunMarkedDesc),
	)
	if err != nil {
		return err
	}

	r.errorCounter, err = r.meter.Int64Counter(
		string(taskRunErrorCountName),
		metric.WithDescription(taskRunErrorCountDesc),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordCountMetrics implements github.com/tektoncd/chains/pkg/metrics.Recorder.RecordCountMetrics
func (r *Recorder) RecordCountMetrics(ctx context.Context, metricType common.Metric) {
	logger := logging.FromContext(ctx)

	if !r.initialized {
		logger.Errorf("ignoring the metrics recording as recorder not initialized")
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch mt := metricType; mt {
	case common.SignedMessagesCount:
		r.signedCounter.Add(ctx, 1)
	case common.PayloadUploadeCount:
		r.uploadedCounter.Add(ctx, 1)
	case common.SignsStoredCount:
		r.storedCounter.Add(ctx, 1)
	case common.MarkedAsSignedCount:
		r.markedCounter.Add(ctx, 1)
	default:
		logger.Errorf("Ignoring the metrics recording as valid Metric type matching %v was not found", mt)
	}
}

// RecordErrorMetric records a TaskRun signing failure with a given error type tag.
func (r *Recorder) RecordErrorMetric(ctx context.Context, errType common.MetricErrorType) {
	if !r.initialized {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.errorCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", string(errType)),
	))
}
