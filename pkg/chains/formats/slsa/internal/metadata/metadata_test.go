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
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)},
				CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, time.UTC)
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)
	want := &slsa.BuildMetadata{
		InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    timestamppb.New(start),
		FinishedOn:   timestamppb.New(end),
	}
	got := GetBuildMetadata(objects.NewTaskRunObjectV1(tr))
	if d := cmp.Diff(want, got, cmp.Options{protocmp.Transform()}); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}

func TestMetadataInTimeZone(t *testing.T) {
	tz := time.FixedZone("Test Time", int((12 * time.Hour).Seconds()))
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-taskrun",
			Namespace: "my-namespace",
			Annotations: map[string]string{
				"chains.tekton.dev/reproducible": "true",
			},
			UID: "abhhf-12354-asjsdbjs23-3435353n",
		},
		Status: v1.TaskRunStatus{
			TaskRunStatusFields: v1.TaskRunStatusFields{
				StartTime:      &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 12, tz)},
				CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, tz)},
			},
		},
	}
	start := time.Date(1995, time.December, 24, 6, 12, 12, 12, tz).UTC()
	end := time.Date(1995, time.December, 24, 6, 12, 12, 24, tz).UTC()
	want := &slsa.BuildMetadata{
		InvocationId: "abhhf-12354-asjsdbjs23-3435353n",
		StartedOn:    timestamppb.New(start),
		FinishedOn:   timestamppb.New(end),
	}
	got := GetBuildMetadata(objects.NewTaskRunObjectV1(tr))
	if d := cmp.Diff(want, got, cmp.Options{protocmp.Transform()}); d != "" {
		t.Fatalf("metadata (-want, +got):\n%s", d)
	}
}
