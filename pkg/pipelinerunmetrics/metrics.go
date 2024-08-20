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

	"github.com/tektoncd/chains/pkg/chains"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/metrics/metricskey"
)

var (
	sgCount = stats.Float64(chains.PipelineRunSignedName,
		chains.PipelineRunSignedDesc,
		stats.UnitDimensionless)

	sgCountView *view.View

	plCount = stats.Float64(chains.PipelineRunUploadedName,
		chains.PipelineRunUploadedDesc,
		stats.UnitDimensionless)

	plCountView *view.View

	stCount = stats.Float64(chains.PipelineRunStoredName,
		chains.PipelineRunStoredDesc,
		stats.UnitDimensionless)

	stCountView *view.View

	mrCount = stats.Float64(chains.PipelineRunMarkedName,
		chains.PipelineRunMarkedDesc,
		stats.UnitDimensionless)

	mrCountView *view.View

	sgCountNS = stats.Float64(chains.PipelineRunSignedMsgPerNamespace,
		chains.PipelineRunSignedMsgDescPerNamespace,
		stats.UnitDimensionless)

	sgCountViewNS *view.View

	plCountNS = stats.Float64(chains.PipelineRunUplPayloadPerNamespace,
		chains.PipelineRunUplPayloadDescPerNamespace,
		stats.UnitDimensionless)

	plCountViewNS *view.View

	stCountNS = stats.Float64(chains.PipelineRunPayloadStoredPerNamespace,
		chains.PipelineRunPayloadStoredDescPerNamespace,
		stats.UnitDimensionless)

	stCountViewNS *view.View

	mrCountNS = stats.Float64(chains.PipelineRunMarkedSignedPerNamespace,
		chains.PipelineRunMarkedDSigneDescPerNamespace,
		stats.UnitDimensionless)

	mrCountViewNS *view.View

	// NamespaceTagKey marks metrics with a namespace.
	NamespaceTagKey = tag.MustNewKey(metricskey.LabelNamespaceName)

	successTagKey = tag.MustNewKey("success")
)

// Recorder holds keys for Tekton metrics
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
// to log the PipelineRun related metrics
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

	sgCountViewNS = &view.View{
		Description: sgCountNS.Description(),
		Measure:     sgCountNS,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{NamespaceTagKey, successTagKey},
	}

	plCountViewNS = &view.View{
		Description: plCountNS.Description(),
		Measure:     plCountNS,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{NamespaceTagKey, successTagKey},
	}

	stCountViewNS = &view.View{
		Description: stCountNS.Description(),
		Measure:     stCountNS,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{NamespaceTagKey, successTagKey},
	}

	mrCountViewNS = &view.View{
		Description: mrCountNS.Description(),
		Measure:     mrCountNS,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{NamespaceTagKey, successTagKey},
	}

	return view.Register(
		sgCountView,
		plCountView,
		stCountView,
		mrCountView,
		sgCountViewNS,
		plCountViewNS,
		stCountViewNS,
		mrCountViewNS,
	)
}

func (r *Recorder) RecordCountMetrics(ctx context.Context, metricType string) {
	logger := logging.FromContext(ctx)
	if !r.initialized {
		logger.Errorf("Ignoring the metrics recording as recorder not initialized ")
		return
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
	case chains.SignedMessagesCountPerNamespace:
		r.countMetrics(ctx, sgCountNS)
	case chains.PayloadUploadeCountPerNamespace:
		r.countMetrics(ctx, plCountNS)
	case chains.SignsStoredCountPerNamespace:
		r.countMetrics(ctx, stCountNS)
	case chains.MarkedAsSignedCountPerNamespace:
		r.countMetrics(ctx, mrCountNS)
	default:
		logger.Errorf("Ignoring the metrics recording as valid Metric type matching %v was not found", mt)
	}

}

func (r *Recorder) countMetrics(ctx context.Context, measure *stats.Float64Measure) {
	metrics.Record(ctx, measure.M(1))
}
