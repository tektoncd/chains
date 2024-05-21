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

package metadata

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetadata(t *testing.T) {
	tests := []struct {
		name     string
		obj      objects.TektonObject
		expected slsa.BuildMetadata
	}{
		{
			name: "taskrun metadata",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					UID: "abhhf-12354-asjsdbjs23-taskrun",
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
						CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
					},
				},
			}),
			expected: slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-taskrun",
				StartedOn:    timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)),
				FinishedOn:   timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)),
			},
		},
		{
			name: "pipelinerun metadata",
			obj: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					UID: "abhhf-12354-asjsdbjs23-pipelinerun",
				},
				Status: v1.PipelineRunStatus{
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
						CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
					},
				},
			}),
			expected: slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-pipelinerun",
				StartedOn:    timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)),
				FinishedOn:   timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)),
			},
		},
	}

	for i := range tests {
		test := &tests[i]
		t.Run(test.name, func(t *testing.T) {
			got := GetBuildMetadata(test.obj)
			if d := cmp.Diff(&test.expected, got, protocmp.Transform()); d != "" {
				t.Fatalf("metadata (-want, +got):\n%s", d)
			}
		})
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))

	tests := []struct {
		name     string
		obj      objects.TektonObject
		expected slsa.BuildMetadata
	}{
		{
			name: "taskrun metadata",
			obj: objects.NewTaskRunObjectV1(&v1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					UID: "abhhf-12354-asjsdbjs23-taskrun",
				},
				Status: v1.TaskRunStatus{
					TaskRunStatusFields: v1.TaskRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
						CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
					},
				},
			}),
			expected: slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-taskrun",
				StartedOn:    timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()),
				FinishedOn:   timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()),
			},
		},
		{
			name: "pipelinerun metadata",
			obj: objects.NewPipelineRunObjectV1(&v1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					UID: "abhhf-12354-asjsdbjs23-pipelinerun",
				},
				Status: v1.PipelineRunStatus{
					PipelineRunStatusFields: v1.PipelineRunStatusFields{
						StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
						CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
					},
				},
			}),
			expected: slsa.BuildMetadata{
				InvocationId: "abhhf-12354-asjsdbjs23-pipelinerun",
				StartedOn:    timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()),
				FinishedOn:   timestamppb.New(time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()),
			},
		},
	}

	for i := range tests {
		test := &tests[i]
		t.Run(test.name, func(t *testing.T) {
			got := GetBuildMetadata(test.obj)
			if d := cmp.Diff(&test.expected, got, protocmp.Transform()); d != "" {
				t.Fatalf("metadata (-want, +got):\n%s", d)
			}
		})
	}
}
