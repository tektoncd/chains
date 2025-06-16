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

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"

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

var (
	sgCount = stats.Float64(string(pipelineRunSignedName),
		pipelineRunSignedDesc,
		stats.UnitDimensionless)

	sgCountView *view.View

	plCount = stats.Float64(string(pipelineRunUploadedName),
		pipelineRunUploadedDesc,
		stats.UnitDimensionless)

	plCountView *view.View

	stCount = stats.Float64(string(pipelineRunStoredName),
		pipelineRunStoredDesc,
		stats.UnitDimensionless)

	stCountView *view.View

	mrCount = stats.Float64(string(pipelineRunMarkedName),
		pipelineRunMarkedDesc,
		stats.UnitDimensionless)

	mrCountView *view.View

	pipelineRunErrorCount = stats.Float64(
		string(pipelineRunErrorCountName),
		pipelineRunErrorCountDesc,
		stats.UnitDimensionless,
	)

	pipelineErrorView *view.View

	errorTypeKey, _ = tag.NewKey("error_type")
)

var _ common.Recorder = &Recorder{}

// Recorder holds keys for PipelineRun metrics.
type Recorder struct {
	initialized bool
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
		r = &Recorder{
			initialized: true,
		}
		errRegistering = viewRegister()
		if errRegistering != nil {
			r.initialized = false
			logger.Errorf("View Register Failed ", r.initialized)
			return
		}
	})

	return r, errRegistering
}

func viewRegister() error {
	sgCountView = &view.View{
		Description: sgCount.Description(),
		Measure:     sgCount,
		Aggregation: view.Count(),
	}

	plCountView = &view.View{
		Description: plCount.Description(),
		Measure:     plCount,
		Aggregation: view.Count(),
	}

	stCountView = &view.View{
		Description: stCount.Description(),
		Measure:     stCount,
		Aggregation: view.Count(),
	}

	mrCountView = &view.View{
		Description: mrCount.Description(),
		Measure:     mrCount,
		Aggregation: view.Count(),
	}

	pipelineErrorView = &view.View{
		Description: pipelineRunErrorCount.Description(),
		Measure:     pipelineRunErrorCount,
		TagKeys:     []tag.Key{errorTypeKey},
		Aggregation: view.Count(),
	}

	return view.Register(
		sgCountView,
		plCountView,
		stCountView,
		mrCountView,
		pipelineErrorView,
	)
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
		r.countMetrics(ctx, sgCount)
	case common.PayloadUploadeCount:
		r.countMetrics(ctx, plCount)
	case common.SignsStoredCount:
		r.countMetrics(ctx, stCount)
	case common.MarkedAsSignedCount:
		r.countMetrics(ctx, mrCount)
	default:
		logger.Errorf("Ignoring the metrics recording as valid Metric type matching %v was not found", mt)
	}
}

func (r *Recorder) countMetrics(ctx context.Context, measure *stats.Float64Measure) {
	metrics.Record(ctx, measure.M(1))
}

// RecordErrorMetric records a PipelineRun signing failure with a given error type tag.
func (r *Recorder) RecordErrorMetric(ctx context.Context, errType common.MetricErrorType) {
	// Add the error_type tag to the context.
	ctx, _ = tag.New(ctx, tag.Upsert(errorTypeKey, string(errType)))
	metrics.Record(ctx, pipelineRunErrorCount.M(1))
}
