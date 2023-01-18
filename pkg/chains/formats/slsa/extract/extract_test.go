/*
Copyright 2022 The Tekton Authors

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

package extract_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

const (
	artifactURL1    = "gcr.io/test/kaniko-chains1"
	artifactDigest1 = "a2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9"
	artifactURL2    = "us-central1-maven.pkg.dev/test/java"
	artifactDigest2 = "b2e500bebfe16cf12fc56316ba72c645e1d29054541dc1ab6c286197434170a9"
)

func TestSubjectDigestsAndRetrieveAllArtifactURIs(t *testing.T) {
	var tests = []struct {
		name string
		// a map of url:digest pairs for type hinting results
		results      map[string]string
		wantSubjects []intoto.Subject
		wantFullURLs []string
	}{
		{
			name: "valid type hinting result fields",
			results: map[string]string{
				artifactURL1: "sha256:" + artifactDigest1,
				artifactURL2: "sha256:" + artifactDigest2,
			},
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL1,
					Digest: map[string]string{
						"sha256": artifactDigest1,
					},
				},
				{
					Name: artifactURL2,
					Digest: map[string]string{
						"sha256": artifactDigest2,
					},
				},
			},
			wantFullURLs: []string{
				fmt.Sprintf("%s@sha256:%s", artifactURL1, artifactDigest1),
				fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest2),
			},
		},
		{
			name: "invalid/missing digest algorithm name",
			results: map[string]string{
				artifactURL1: "sha1:" + artifactDigest1,
				artifactURL2: artifactDigest2,
			},
			wantSubjects: nil,
			wantFullURLs: []string{},
		},
		{
			name: "invalid digest sha",
			results: map[string]string{
				artifactURL1: "sha256:a123",
			},
			wantSubjects: nil,
			wantFullURLs: []string{},
		},
		{
			name: "invalid url value",
			results: map[string]string{
				"": "sha256:" + artifactDigest1,
			},
			wantSubjects: nil,
			wantFullURLs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// test both taskrun object and pipelinerun object
			runObjects := []objects.TektonObject{
				createTaskRunObjectWithResults(tc.results),
				createPipelineRunObjectWithResults(tc.results),
			}

			for _, o := range runObjects {
				gotSubjects := extract.SubjectDigests(o, logtesting.TestLogger(t))
				if diff := cmp.Diff(tc.wantSubjects, gotSubjects, cmpopts.SortSlices(func(x, y intoto.Subject) bool { return x.Name < y.Name })); diff != "" {
					t.Errorf("Wrong subjects extracted, diff=%s", diff)
				}

				gotURIs := extract.RetrieveAllArtifactURIs(o, logtesting.TestLogger(t))
				if diff := cmp.Diff(tc.wantFullURLs, gotURIs, cmpopts.SortSlices(func(x, y string) bool { return x < y })); diff != "" {
					t.Errorf("Wrong URIs extracted, diff=%s", diff)
				}
			}

		})
	}
}

func createTaskRunObjectWithResults(results map[string]string) objects.TektonObject {
	trResults := []v1beta1.TaskRunResult{}
	prefix := 0
	for url, digest := range results {
		trResults = append(trResults,
			v1beta1.TaskRunResult{Name: fmt.Sprintf("%v_IMAGE_DIGEST", prefix), Value: *v1beta1.NewStructuredValues(digest)},
			v1beta1.TaskRunResult{Name: fmt.Sprintf("%v_IMAGE_URL", prefix), Value: *v1beta1.NewStructuredValues(url)},
		)
		prefix++
	}

	return objects.NewTaskRunObject(
		&v1beta1.TaskRun{
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					TaskRunResults: trResults,
				},
			},
		},
	)
}

func createPipelineRunObjectWithResults(results map[string]string) objects.TektonObject {
	prResults := []v1beta1.PipelineRunResult{}
	prefix := 0
	for url, digest := range results {
		prResults = append(prResults,
			v1beta1.PipelineRunResult{Name: fmt.Sprintf("%v_IMAGE_DIGEST", prefix), Value: *v1beta1.NewStructuredValues(digest)},
			v1beta1.PipelineRunResult{Name: fmt.Sprintf("%v_IMAGE_URL", prefix), Value: *v1beta1.NewStructuredValues(url)},
		)
		prefix++
	}

	return objects.NewPipelineRunObject(
		&v1beta1.PipelineRun{
			Status: v1beta1.PipelineRunStatus{
				PipelineRunStatusFields: v1beta1.PipelineRunStatusFields{
					PipelineResults: prResults,
				},
			},
		},
	)
}
