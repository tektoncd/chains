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
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/extract"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/compare"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			ctx := logtesting.TestContextWithLogger(t)
			// test both taskrun object and pipelinerun object
			runObjects := []objects.TektonObject{
				createTaskRunObjectWithResults(tc.results),
				createProWithPipelineResults(tc.results),
			}
			for _, o := range runObjects {
				gotSubjects := extract.SubjectDigests(ctx, o, &slsaconfig.SlsaConfig{DeepInspectionEnabled: false})
				if diff := cmp.Diff(tc.wantSubjects, gotSubjects, compare.SubjectCompareOption()); diff != "" {
					t.Errorf("Wrong subjects extracted, diff=%s", diff)
				}

				gotURIs := extract.RetrieveAllArtifactURIs(ctx, o, false)
				if diff := cmp.Diff(tc.wantFullURLs, gotURIs, cmpopts.SortSlices(func(x, y string) bool { return x < y })); diff != "" {
					t.Errorf("Wrong URIs extracted, diff=%s", diff)
				}
			}

		})
	}
}

func TestPipelineRunObserveModeForSubjects(t *testing.T) {
	var tests = []struct {
		name                  string
		pro                   objects.TektonObject
		deepInspectionEnabled bool
		wantSubjects          []intoto.Subject
		wantFullURLs          []string
	}{
		{
			name:                  "deep inspection disabled",
			pro:                   createProWithPipelineResults(map[string]string{artifactURL1: "sha256:" + artifactDigest1}),
			deepInspectionEnabled: false,
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL1,
					Digest: map[string]string{
						"sha256": artifactDigest1,
					},
				},
			},
			wantFullURLs: []string{fmt.Sprintf("%s@sha256:%s", artifactURL1, artifactDigest1)},
		},
		{
			name:                  "deep inspection enabled: no duplication",
			pro:                   createProWithTaskRunResults(nil, []artifact{{uri: artifactURL2, digest: "sha256:" + artifactDigest2}}),
			deepInspectionEnabled: true,
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL2,
					Digest: map[string]string{
						"sha256": artifactDigest2,
					},
				},
			},
			wantFullURLs: []string{fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest2)},
		},
		{
			name: "deep inspection enabled: 2 tasks have same uri with different sha256 digests",
			pro: createProWithTaskRunResults(nil, []artifact{
				{uri: artifactURL2, digest: "sha256:" + artifactDigest1},
				{uri: artifactURL2, digest: "sha256:" + artifactDigest2},
			}),
			deepInspectionEnabled: true,
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL2,
					Digest: map[string]string{
						"sha256": artifactDigest2,
					},
				},
				{
					Name: artifactURL2,
					Digest: map[string]string{
						"sha256": artifactDigest1,
					},
				},
			},
			wantFullURLs: []string{
				fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest1),
				fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest2),
			},
		},
		{
			name: "deep inspection enabled: 2 taskruns have same uri with same sha256 digests",
			pro: createProWithTaskRunResults(nil, []artifact{
				{uri: artifactURL2, digest: "sha256:" + artifactDigest2},
				{uri: artifactURL2, digest: "sha256:" + artifactDigest2},
			}),
			deepInspectionEnabled: true,
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL2,
					Digest: map[string]string{
						"sha256": artifactDigest2,
					},
				},
			},
			wantFullURLs: []string{
				fmt.Sprintf("%s@sha256:%s", artifactURL2, artifactDigest2),
			},
		},
		{
			name: "deep inspection enabled: pipelinerun and taskrun have duplicated results",
			pro: createProWithTaskRunResults(
				createProWithPipelineResults(map[string]string{artifactURL1: "sha256:" + artifactDigest1}).(*objects.PipelineRunObjectV1),
				[]artifact{
					{uri: artifactURL1, digest: "sha256:" + artifactDigest1},
				}),
			deepInspectionEnabled: true,
			wantSubjects: []intoto.Subject{
				{
					Name: artifactURL1,
					Digest: map[string]string{
						"sha256": artifactDigest1,
					},
				},
			},
			wantFullURLs: []string{
				fmt.Sprintf("%s@sha256:%s", artifactURL1, artifactDigest1),
			},
		},
		{
			name: "deep inspection enabled: pipelinerun and taskrun have different results",
			pro: createProWithTaskRunResults(
				createProWithPipelineResults(map[string]string{artifactURL1: "sha256:" + artifactDigest1}).(*objects.PipelineRunObjectV1),
				[]artifact{
					{uri: artifactURL2, digest: "sha256:" + artifactDigest2},
				}),
			deepInspectionEnabled: true,
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)

			gotSubjects := extract.SubjectDigests(ctx, tc.pro, &slsaconfig.SlsaConfig{DeepInspectionEnabled: tc.deepInspectionEnabled})
			if diff := cmp.Diff(tc.wantSubjects, gotSubjects, compare.SubjectCompareOption()); diff != "" {
				t.Errorf("Wrong subjects extracted, diff=%s, %s", diff, gotSubjects)
			}

			gotURIs := extract.RetrieveAllArtifactURIs(ctx, tc.pro, tc.deepInspectionEnabled)
			if diff := cmp.Diff(tc.wantFullURLs, gotURIs, cmpopts.SortSlices(func(x, y string) bool { return x < y })); diff != "" {
				t.Errorf("Wrong URIs extracted, diff=%s", diff)
			}
		})
	}
}

func TestSubjectsFromBuildArtifact(t *testing.T) {
	tests := []struct {
		name             string
		results          []objects.Result
		expectedSubjects []intoto.Subject
	}{
		{
			name: "no type-hinted build artifacts",
			results: []objects.Result{
				{
					Name:  "result2_ARTIFACT_URL",
					Value: *v1.NewStructuredValues("gcr.io/test/img4"),
				},
				{
					Name:  "result2_ARTIFACT_DIGEST",
					Value: *v1.NewStructuredValues("sha256:jh5f72309sf937b16a914eea7cb81ebaa8f2b0a13833797acd26dty46529ihm"),
				},
				{
					Name: "result3_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":    "gcr.io/img5",
						"digest": "sha256:7492314e32aa75ff1f2cfea35b7dda85d8831929d076aab52420c3400c8c65d8",
					}),
				},
			},
		},
		{
			name: "type-hinted build artifacts",
			results: []objects.Result{
				{
					Name:  "result1_IMAGE_URL",
					Value: *v1.NewStructuredValues("gcr.io/test/img1"),
				},
				{
					Name:  "result1_IMAGE_DIGEST",
					Value: *v1.NewStructuredValues("sha256:52e18b100a8da6e191a1955913ba127b75a8b38146cd9b0f573ec1d8e8ecd135"),
				},
				{
					Name: "IMAGES",
					Value: *v1.NewStructuredValues(
						"gcr.io/test/img2@sha256:2996854378975c2f8011ddf0526975d1aaf1790b404da7aad4bf25293055bc8b, " +
							"gcr.io/test/img3@sha256:ef334b5d9704da9b325ed6d4e3e5327863847e2da6d43f81831fd1decbdb2213",
					),
				},
				{
					Name: "result2_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":             "gcr.io/test/img4",
						"digest":          "sha256:910700c5ace59f70588c4e2a38ed131146c9f65c94379dfe12376075fc2f338f",
						"isBuildArtifact": "true",
					}),
				},
				{
					Name: "result3_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":             "gcr.io/test/img5",
						"digest":          "sha256:7492314e32aa75ff1f2cfea35b7dda85d8831929d076aab52420c3400c8c65d8",
						"isBuildArtifact": "true",
					}),
				},
			},
			expectedSubjects: []intoto.Subject{
				{
					Name: "gcr.io/test/img4",
					Digest: map[string]string{
						"sha256": "910700c5ace59f70588c4e2a38ed131146c9f65c94379dfe12376075fc2f338f",
					},
				},
				{
					Name: "gcr.io/test/img5",
					Digest: map[string]string{
						"sha256": "7492314e32aa75ff1f2cfea35b7dda85d8831929d076aab52420c3400c8c65d8",
					},
				},
				{
					Name: "gcr.io/test/img1",
					Digest: map[string]string{
						"sha256": "52e18b100a8da6e191a1955913ba127b75a8b38146cd9b0f573ec1d8e8ecd135",
					},
				},
				{
					Name: "gcr.io/test/img2",
					Digest: map[string]string{
						"sha256": "2996854378975c2f8011ddf0526975d1aaf1790b404da7aad4bf25293055bc8b",
					},
				},
				{
					Name: "gcr.io/test/img3",
					Digest: map[string]string{
						"sha256": "ef334b5d9704da9b325ed6d4e3e5327863847e2da6d43f81831fd1decbdb2213",
					},
				},
			},
		},
		{
			name: "no repetead type-hinted build artifacts",
			results: []objects.Result{
				{
					Name: "result1_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":             "gcr.io/test/img1",
						"digest":          "sha256:8b7b3e8b124f937b16a914eea7cb81ebaa8f2b0a13833797acd26d67edf4e056",
						"isBuildArtifact": "true",
					}),
				},
				{
					Name: "result2_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":             "gcr.io/test/img1",
						"digest":          "sha256:8b7b3e8b124f937b16a914eea7cb81ebaa8f2b0a13833797acd26d67edf4e056",
						"isBuildArtifact": "true",
					}),
				},
				{
					Name: "IMAGES",
					Value: *v1.NewStructuredValues(
						"gcr.io/test/img1@sha256:8b7b3e8b124f937b16a914eea7cb81ebaa8f2b0a13833797acd26d67edf4e056",
					),
				},
			},
			expectedSubjects: []intoto.Subject{
				{
					Name: "gcr.io/test/img1",
					Digest: map[string]string{
						"sha256": "8b7b3e8b124f937b16a914eea7cb81ebaa8f2b0a13833797acd26d67edf4e056",
					},
				},
			},
		},
		{
			name: "malformed digests",
			results: []objects.Result{
				{
					Name:  "result1_IMAGE_URL",
					Value: *v1.NewStructuredValues("gcr.io/test/img1"),
				},
				{
					Name:  "result1_IMAGE_DIGEST",
					Value: *v1.NewStructuredValues("sha256@52e18b100a8da6e191a1955913ba127b75a8b38146cd9b0f573ec1d8e8ecd135"),
				},
				{
					Name: "IMAGES",
					Value: *v1.NewStructuredValues(
						"gcr.io/test/img2@sha256@2996854378975c2f8011ddf0526975d1aaf1790b404da7aad4bf25293055bc8b",
					),
				},
				{
					Name: "result2_ARTIFACT_OUTPUTS",
					Value: *v1.NewObject(map[string]string{
						"uri":             "gcr.io/test/img5",
						"digest":          "sha256@7492314e32aa75ff1f2cfea35b7dda85d8831929d076aab52420c3400c8c65d8",
						"isBuildArtifact": "true",
					}),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			got := extract.SubjectsFromBuildArtifact(ctx, test.results)
			if diff := cmp.Diff(test.expectedSubjects, got); diff != "" {
				t.Errorf("Wrong subjects from build artifacts, +got -want, diff=%s", diff)
			}
		})
	}
}

func createTaskRunObjectWithResults(results map[string]string) objects.TektonObject {
	trResults := []v1.TaskRunResult{}
	prefix := 0
	for url, digest := range results {
		trResults = append(trResults,
			v1.TaskRunResult{Name: fmt.Sprintf("%v_IMAGE_DIGEST", prefix), Value: *v1.NewStructuredValues(digest)},
			v1.TaskRunResult{Name: fmt.Sprintf("%v_IMAGE_URL", prefix), Value: *v1.NewStructuredValues(url)},
		)
		prefix++
	}

	return objects.NewTaskRunObjectV1(
		&v1.TaskRun{
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					Results: trResults,
				},
			},
		},
	)
}

func createProWithPipelineResults(results map[string]string) objects.TektonObject {
	prResults := []v1.PipelineRunResult{}
	prefix := 0
	for url, digest := range results {
		prResults = append(prResults,
			v1.PipelineRunResult{Name: fmt.Sprintf("%v_IMAGE_DIGEST", prefix), Value: *v1.NewStructuredValues(digest)},
			v1.PipelineRunResult{Name: fmt.Sprintf("%v_IMAGE_URL", prefix), Value: *v1.NewStructuredValues(url)},
		)
		prefix++
	}

	return objects.NewPipelineRunObjectV1(
		&v1.PipelineRun{
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					Results: prResults,
				},
			},
		},
	)
}

type artifact struct {
	uri    string
	digest string
}

// create a child taskrun for each result
//
//nolint:all
func createProWithTaskRunResults(pro *objects.PipelineRunObjectV1, results []artifact) objects.TektonObject {
	if pro == nil {
		pro = objects.NewPipelineRunObjectV1(&v1.PipelineRun{
			Status: v1.PipelineRunStatus{
				PipelineRunStatusFields: v1.PipelineRunStatusFields{
					PipelineSpec: &v1.PipelineSpec{},
				},
			},
		})
	}

	if pro.Status.PipelineSpec == nil {
		pro.Status.PipelineSpec = &v1.PipelineSpec{}
	}

	// create child taskruns with results and pipelinetask
	prefix := 0
	for _, r := range results {
		// simulate child taskruns
		pipelineTaskName := fmt.Sprintf("task-%d", prefix)
		tr := &v1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{objects.PipelineTaskLabel: pipelineTaskName}},
			Status: v1.TaskRunStatus{
				TaskRunStatusFields: v1.TaskRunStatusFields{
					CompletionTime: &metav1.Time{Time: time.Date(1995, time.December, 24, 6, 12, 12, 24, time.UTC)},
					Results: []v1.TaskRunResult{
						{Name: fmt.Sprintf("%v_IMAGE_DIGEST", prefix), Value: *v1.NewStructuredValues(r.digest)},
						{Name: fmt.Sprintf("%v_IMAGE_URL", prefix), Value: *v1.NewStructuredValues(r.uri)},
					},
				},
			},
		}

		pro.AppendTaskRun(tr)
		pro.Status.PipelineSpec.Tasks = append(pro.Status.PipelineSpec.Tasks, v1.PipelineTask{Name: pipelineTaskName})
		prefix++
	}

	return pro
}
