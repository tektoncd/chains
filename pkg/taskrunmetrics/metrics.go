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

	"github.com/tektoncd/chains/pkg/chains"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
)

var (
	sgCountView *view.View

	sgCount = stats.Float64(chains.TaskRunSignedName,
		chains.TaskRunSignedDesc,
		stats.UnitDimensionless)

	plCount = stats.Float64(chains.TaskRunUploadedName,
		chains.TaskRunUploadedDesc,
		stats.UnitDimensionless)

	plCountView *view.View

	stCount = stats.Float64(chains.TaskRunStoredName,
		chains.TaskRunStoredDesc,
		stats.UnitDimensionless)

	stCountView *view.View

	mrCount = stats.Float64(chains.TaskRunMarkedName,
		chains.TaskRunMarkedDesc,
		stats.UnitDimensionless)

	mrCountView *view.View
)

// Recorder is used to actually record TaskRun metrics
type Recorder struct {
	initialized bool
}

// We cannot register the view multiple times, so NewRecorder lazily
// initializes this singleton and returns the same recorder across any
// subsequent invocations.
var (
	once           sync.Once
	r              *Recorder
	errRegistering error
)

// NewRecorder creates a new metrics recorder instance
// to log the TaskRun related metrics
func NewRecorder(ctx context.Context) (*Recorder, error) {
	once.Do(func() {
		r = &Recorder{
			initialized: true,
		}

		errRegistering = viewRegister()

		if errRegistering != nil {
			r.initialized = false
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
	return view.Register(
		sgCountView,
		plCountView,
		stCountView,
		mrCountView,
	)
}

func (r *Recorder) RecordCountMetrics(ctx context.Context, metricType string) {
	logger := logging.FromContext(ctx)

	if !r.initialized {
		logger.Errorf("ignoring the metrics recording as recorder not initialized ")
	}
	switch mt := metricType; mt {
	case chains.SignedMessagesCount:
		r.countMetrics(ctx, sgCount)
	case chains.PayloadUploadeCount:
		r.countMetrics(ctx, plCount)
	case chains.SignsStoredCount:
		r.countMetrics(ctx, stCount)
	case chains.MarkedAsSignedCount:
		r.countMetrics(ctx, mrCount)
	default:
		logger.Errorf("Ignoring the metrics recording as valid Metric type matching %v was not found", mt)
	}

}

func (r *Recorder) countMetrics(ctx context.Context, measure *stats.Float64Measure) {
	metrics.Record(ctx, measure.M(1))
}
