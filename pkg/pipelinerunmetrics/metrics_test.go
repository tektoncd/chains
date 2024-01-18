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

	"github.com/tektoncd/chains/pkg/chains"
	"knative.dev/pkg/metrics/metricstest"
	_ "knative.dev/pkg/metrics/testing"
)

func TestUninitializedMetrics(t *testing.T) {
	metrics := &Recorder{}
	ctx := context.Background()

	metrics.RecordCountMetrics(ctx, chains.SignedMessagesCount)
	metricstest.CheckStatsNotReported(t, chains.PipelineRunSignedName)

	metrics.RecordCountMetrics(ctx, chains.PayloadUploadeCount)
	metricstest.CheckStatsNotReported(t, chains.PipelineRunUploadedName)

	metrics.RecordCountMetrics(ctx, chains.SignsStoredCount)
	metricstest.CheckStatsNotReported(t, chains.PipelineRunStoredName)

	metrics.RecordCountMetrics(ctx, chains.MarkedAsSignedCount)
	metricstest.CheckStatsNotReported(t, chains.PipelineRunMarkedName)
}

func TestCountMetrics(t *testing.T) {
	unregisterMetrics()
	ctx := context.Background()
	ctx = WithClient(ctx)

	rec := Get(ctx)

	rec.RecordCountMetrics(ctx, chains.SignedMessagesCount)
	metricstest.CheckCountData(t, chains.PipelineRunSignedName, map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, chains.PayloadUploadeCount)
	metricstest.CheckCountData(t, chains.PipelineRunUploadedName, map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, chains.SignsStoredCount)
	metricstest.CheckCountData(t, chains.PipelineRunStoredName, map[string]string{}, 1)
	rec.RecordCountMetrics(ctx, chains.MarkedAsSignedCount)
	metricstest.CheckCountData(t, chains.PipelineRunMarkedName, map[string]string{}, 1)
}

func unregisterMetrics() {
	metricstest.Unregister(chains.PipelineRunSignedName, chains.PipelineRunUploadedName, chains.PipelineRunStoredName, chains.PipelineRunMarkedName)
	// Allow the recorder singleton to be recreated.
	once = sync.Once{}
	r = nil
}
