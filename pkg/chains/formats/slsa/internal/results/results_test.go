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

package results

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/attestation/go/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestGetResultsWithoutBuildArtifacts(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		objName  string
		results  []objects.Result
		expected []*slsa.ResourceDescriptor
	}{
		{
			name:     "no results as input",
			expected: []*slsa.ResourceDescriptor{},
		},
		{
			name:    "results without build artifacts",
			prefix:  "taskRunResults/%s/%s",
			objName: "taskRun-name",
			results: []objects.Result{
				{
					Name: "result1",
					Type: v1.ResultsTypeString,
					Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "my-first-result",
					},
				},
				{
					Name: "res2-ARTIFACT_URI",
					Type: v1.ResultsTypeString,
					Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "gcr.io/my/image/fromstep2",
					},
				},
				{
					Name: "res2-ARTIFACT_DIGEST",
					Type: v1.ResultsTypeString,
					Value: v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					},
				},
				{
					Name: "res3-ARTIFACT_OUTPUTS",
					Type: v1.ResultsTypeObject,
					Value: v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":    "oci://gcr.io/test1/test1",
							"digest": "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
						},
					},
				},
				{
					Name: "res4-ARTIFACT_OUTPUTS",
					Type: v1.ResultsTypeObject,
					Value: v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":             "git+https://github.com/test",
							"digest":          "sha1:ab123",
							"isBuildArtifact": "true",
						},
					},
				},
				{
					Name: "res5-ARTIFACT_OUTPUTS",
					Type: v1.ResultsTypeObject,
					Value: v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":             "oci://gcr.io/test2/test2",
							"digest":          "sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
							"isBuildArtifact": "false",
						},
					},
				},
				{
					Name: "res6-ARTIFACT_OUTPUTS",
					Type: v1.ResultsTypeObject,
					Value: v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"digest":          "sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
							"isBuildArtifact": "true",
						},
					},
				},
				{
					Name: "res7-ARTIFACT_OUTPUTS",
					Type: v1.ResultsTypeObject,
					Value: v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":             "oci://gcr.io/test2/test2",
							"digest":          "sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
							"isBuildArtifact": "true",
						},
					},
				},
			},
			expected: []*slsa.ResourceDescriptor{
				{
					Name:      "taskRunResults/taskRun-name/result1",
					MediaType: "application/json",
					Content: toJSONString(t, v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "my-first-result",
					}),
				},
				{
					Name:      "taskRunResults/taskRun-name/res2-ARTIFACT_URI",
					MediaType: "application/json",
					Content: toJSONString(t, v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "gcr.io/my/image/fromstep2",
					}),
				},
				{
					Name:      "taskRunResults/taskRun-name/res2-ARTIFACT_DIGEST",
					MediaType: "application/json",
					Content: toJSONString(t, v1.ParamValue{
						Type:      v1.ParamTypeString,
						StringVal: "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
					}),
				},
				{
					Name:      "taskRunResults/taskRun-name/res3-ARTIFACT_OUTPUTS",
					MediaType: "application/json",
					Content: toJSONString(t, v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":    "oci://gcr.io/test1/test1",
							"digest": "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
						},
					}),
				},
				{
					Name:      "taskRunResults/taskRun-name/res5-ARTIFACT_OUTPUTS",
					MediaType: "application/json",
					Content: toJSONString(t, v1.ParamValue{
						Type: v1.ParamTypeObject,
						ObjectVal: map[string]string{
							"uri":             "oci://gcr.io/test2/test2",
							"digest":          "sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
							"isBuildArtifact": "false",
						},
					}),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetResultsWithoutBuildArtifacts(test.objName, test.results, test.prefix)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if d := cmp.Diff(test.expected, got, cmp.Options{protocmp.Transform()}); d != "" {
				t.Fatalf("metadata (-want, +got):\n%s", d)
			}
		})
	}
}

func toJSONString(t *testing.T, val v1.ParamValue) []byte {
	t.Helper()
	res, err := json.Marshal(val)
	if err != nil {
		t.Fatalf("error converting to json string: %v", err)
	}

	return res
}
