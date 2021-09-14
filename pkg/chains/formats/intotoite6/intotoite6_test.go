/*
Copyright 2021 The Tekton Authors

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

package intotoite6

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"

	"github.com/google/go-cmp/cmp"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var testData1 = `
{
  "spec": {
    "params": [
      {
        "name": "IMAGE",
        "value": "test.io/test/image"
      },
      {
        "name": "CHAINS-GIT_COMMIT",
        "value": "abcd"
      },
      {
        "name": "CHAINS-GIT_URL",
        "value": "https://git.test.com"
      },
      {
        "name": "filename",
        "value": "/bin/ls"
      }
    ],
    "taskRef": {
      "name": "test-task",
      "kind": "Task"
    },
    "serviceAccountName": "default"
  },
  "status": {
    "startTime": "2021-03-29T09:50:00Z",
    "completionTime": "2021-03-29T09:50:15Z",
    "conditions": [
      {
        "type": "Succeeded",
        "status": "True",
        "lastTransitionTime": "2021-03-29T09:50:15Z",
        "reason": "Succeeded",
        "message": "All Steps have completed executing"
      }
    ],
    "podName": "test-pod-name",
    "steps": [
      {
        "name": "step1",
        "container": "step-step1",
        "imageID": "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"
      },
      {
        "name": "step2",
        "container": "step-step2",
        "imageID": "docker-pullable://gcr.io/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac"
      },
      {
        "name": "step3",
        "container": "step-step3",
        "imageID": "docker-pullable://gcr.io/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478"
      }
    ],
    "taskResults": [
      {
        "name": "IMAGE_DIGEST",
        "value": "sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7"
      },
      {
        "name": "filename_DIGEST",
        "value": "sha256:hash5   /bin/ls"
      }
    ],
    "taskSpec": {
      "params": [
        {
          "name": "IMAGE",
          "type": "string"
        },
        {
          "name": "filename",
          "type": "string"
        },
        {
          "name": "DOCKERFILE",
          "type": "string"
        },
        {
          "name": "CONTEXT",
          "type": "string"
        },
        {
          "name": "EXTRA_ARGS",
          "type": "string"
        },
        {
          "name": "BUILDER_IMAGE",
          "type": "string"
        }
      ],
      "results": [
        {
          "name": "IMAGE_DIGEST",
          "description": "Digest of the image just built."
        },
        {
          "name": "filename_DIGEST",
          "description": "Digest of the file just built."
        }
      ]
    }
  }
}
`

var e1BuildStart = time.Unix(1617011400, 0)
var e1BuildFinished = time.Unix(1617011415, 0)
var expected1 = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "/bin/ls",
				Digest: map[string]string{
					"sha256": "hash5",
				},
			},
			{
				Name: "pkg:docker/test/image@sha256:827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7?repository_url=test.io",
				Digest: map[string]string{
					"sha256": "827521c857fdcd4374f4da5442fbae2edb01e7fbae285c3ec15673d4c1daecb7",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{
			BuildStartedOn:  &e1BuildStart,
			BuildFinishedOn: &e1BuildFinished,
		},
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "git+https://git.test.com",
				Digest: map[string]string{
					"git_commit": "abcd",
				},
			},
			{
				URI: "pkg:docker/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6?repository_url=gcr.io",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
			{
				URI: "pkg:docker/test2/test2@sha256:4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac?repository_url=gcr.io",
				Digest: map[string]string{
					"sha256": "4d6dd704ef58cb214dd826519929e92a978a57cdee43693006139c0080fd6fac",
				},
			},
			{
				URI: "pkg:docker/test3/test3@sha256:f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478?repository_url=gcr.io",
				Digest: map[string]string{
					"sha256": "f1a8b8549c179f41e27ff3db0fe1a1793e4b109da46586501a8343637b1d0478",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-1",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:              tektonID,
			EntryPoint:        "test-task",
			DefinedInMaterial: intP(0),
		},
	},
}

var testData2 = `
{
  "spec": {
    "params": [],
    "taskRef": {
      "name": "test-task",
      "kind": "Task"
    },
    "serviceAccountName": "default"
  },
  "status": {
    "conditions": [
      {
        "type": "Succeeded",
        "status": "True",
        "lastTransitionTime": "2021-03-29T09:50:15Z",
        "reason": "Succeeded",
        "message": "All Steps have completed executing"
      }
    ],
    "podName": "test-pod-name",
    "steps": [
      {
        "name": "step1",
        "container": "step-step1",
        "imageID": "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"
      }
    ],
    "taskResults": [
      {
        "name": "some-uri_DIGEST",
        "value": "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"
      },
      {
        "name": "some-uri",
        "value": "pkg:deb/debian/curl@7.50.3-1"
      }
    ],
    "taskSpec": {
      "params": [],
      "results": [
        {
          "name": "some-uri_DIGEST",
          "description": "Digest of a file to push."
        },
        {
          "name": "some-uri",
          "description": "some calculated uri"
        }
      ]
    }
  }
}
`

var expected2 = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "pkg:deb/debian/curl@7.50.3-1",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{},
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "pkg:docker/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6?repository_url=gcr.io",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-2",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:       tektonID,
			EntryPoint: "test-task",
		},
	},
}

var dataMultpleSubjects = `
{
  "spec": {
    "params": [],
    "taskRef": {
      "name": "test-task",
      "kind": "Task"
    },
    "serviceAccountName": "default"
  },
  "status": {
    "conditions": [
      {
        "type": "Succeeded",
        "status": "True",
        "lastTransitionTime": "2021-03-29T09:50:15Z",
        "reason": "Succeeded",
        "message": "All Steps have completed executing"
      }
    ],
    "podName": "test-pod-name",
    "steps": [
      {
        "name": "step1",
        "container": "step-step1",
        "imageID": "docker-pullable://gcr.io/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"
      }
    ],
    "taskResults": [
      {
        "name": "file1_DIGEST",
        "value": "sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6"
      },
      {
        "name": "file1",
        "value": "pkg:deb/debian/curl@7.50.3-1"
      },
      {
        "name": "file2_DIGEST",
        "value": "sha256:daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367"
      },
      {
        "name": "file2",
        "value": "pkg:generic/curl@7.50.3-1.tar.gz"
      }
    ],
    "taskSpec": {
      "params": [],
      "results": [
        {
          "name": "file1_DIGEST",
          "description": "Digest of a file to push."
        },
        {
          "name": "file1",
          "description": "some assembled file"
        },
        {
          "name": "file2_DIGEST",
          "description": "Digest of a file to push."
        },
        {
          "name": "file2",
          "description": "some assembled file"
        }
      ]
    }
  }
}
`

var expectedMultipleSubjects = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateSLSAProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "pkg:deb/debian/curl@7.50.3-1",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
			{
				Name: "pkg:generic/curl@7.50.3-1.tar.gz",
				Digest: map[string]string{
					"sha256": "daa1a56e13c85cf164e7d9e595006649e3a04c47fe4a8261320e18a0bf3b0367",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Metadata: &in_toto.ProvenanceMetadata{},
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "pkg:docker/test1/test1@sha256:d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6?repository_url=gcr.io",
				Digest: map[string]string{
					"sha256": "d4b63d3e24d6eef04a6dc0795cf8a73470688803d97c52cffa3c8d4efd3397b6",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test_builder-multiple",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:       tektonID,
			EntryPoint: "test-task",
		},
	},
}

func TestInTotoIte6_CreatePayload1(t *testing.T) {
	var tr v1beta1.TaskRun

	err := json.Unmarshal([]byte(testData1), &tr)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-1",
		},
	}
	i, _ := NewFormatter(cfg)

	got, err := i.CreatePayload(&tr)

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected1, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestInTotoIte6_CreatePayload2(t *testing.T) {
	var tr v1beta1.TaskRun

	err := json.Unmarshal([]byte(testData2), &tr)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-2",
		},
	}
	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(&tr)

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expected2, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestInTotoIte6_CreatePayloadNilTaskRef(t *testing.T) {
	var tr v1beta1.TaskRun

	if err := json.Unmarshal([]byte(testData1), &tr); err != nil {
		t.Fatal(err)
	}

	tr.Spec.TaskRef = nil
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg)

	p, err := f.CreatePayload(&tr)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}

	ps := p.(in_toto.ProvenanceStatement)
	if diff := cmp.Diff(tr.Name, ps.Predicate.Recipe.EntryPoint); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestInTotoIte6_MultipleSubjects(t *testing.T) {
	var tr v1beta1.TaskRun

	err := json.Unmarshal([]byte(dataMultpleSubjects), &tr)
	if err != nil {
		t.Errorf("json.Unmarshal() error = %v", err)
		return
	}

	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "test_builder-multiple",
		},
	}
	i, _ := NewFormatter(cfg)
	got, err := i.CreatePayload(&tr)

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if diff := cmp.Diff(expectedMultipleSubjects, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestNewFormatter(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		cfg := config.Config{
			Builder: config.BuilderConfig{
				ID: "testid",
			},
		}
		f, err := NewFormatter(cfg)
		if f == nil {
			t.Error("Failed to create formatter")
		}
		if err != nil {
			t.Errorf("Error creating formatter: %s", err)
		}
	})
}

func TestPurlDocker(t *testing.T) {
	tests := []struct {
		imageID string
		purl    string
		alg     string
		digest  string
	}{
		{
			imageID: "alpine@sha256:3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
			purl:    "pkg:docker/alpine@sha256:3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
			alg:     "sha256",
			digest:  "3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
		},
		{
			imageID: "org/alpine@sha256:3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
			purl:    "pkg:docker/org/alpine@sha256:3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
			alg:     "sha256",
			digest:  "3f1017b520fe358d7b3796879232cd36259066ccd5bab5466cbedb444064dfed",
		},
		{
			imageID: "docker://alpine@sha256:6e0447537050cf871f9ab6a3fec5715f9c6fff5212f6666993f1fc46b1f717a3",
			purl:    "pkg:docker/alpine@sha256:6e0447537050cf871f9ab6a3fec5715f9c6fff5212f6666993f1fc46b1f717a3",
			alg:     "sha256",
			digest:  "6e0447537050cf871f9ab6a3fec5715f9c6fff5212f6666993f1fc46b1f717a3",
		},
		{
			imageID: "docker://org/name@sha256:55bbe28f6e4abb21be67cdd592e6e3b7b21b1a7b159768d539eb63119bbc1d28",
			purl:    "pkg:docker/org/name@sha256:55bbe28f6e4abb21be67cdd592e6e3b7b21b1a7b159768d539eb63119bbc1d28",
			alg:     "sha256",
			digest:  "55bbe28f6e4abb21be67cdd592e6e3b7b21b1a7b159768d539eb63119bbc1d28",
		},
		{
			imageID: "docker://gcr.io/org/name@sha256:64e1a1f5bd1c888e107e0145e26582edfab24779c1bbb0e11f3768432c5c0399",
			purl:    "pkg:docker/org/name@sha256:64e1a1f5bd1c888e107e0145e26582edfab24779c1bbb0e11f3768432c5c0399?repository_url=gcr.io",
			alg:     "sha256",
			digest:  "64e1a1f5bd1c888e107e0145e26582edfab24779c1bbb0e11f3768432c5c0399",
		},
	}

	for _, test := range tests {
		purl, alg, digest := getPackageURLDocker(test.imageID)
		if purl != test.purl {
			t.Errorf("Invalid package url, got '%s' want '%s'", purl, test.purl)
		}
		if alg != test.alg {
			t.Errorf("Invalid alg, got '%s' want '%s'", alg, test.alg)
		}
		if digest != test.digest {
			t.Errorf("Invalid digest, got '%s' want '%s'", digest, test.digest)
		}
	}
}

func TestGetOCIImageID(t *testing.T) {
	tests := []struct {
		imageID string
		name    string
		alg     string
		digest  string
	}{
		{
			imageID: "docker://name@sha256:digest",
			name:    "name",
			alg:     "sha256",
			digest:  "digest",
		},
	}
	for _, test := range tests {
		imageID := getOCIImageID(test.name, test.alg, test.digest)
		if imageID != test.imageID {
			t.Errorf("Invalid image ID, got '%s' want '%s'", imageID, test.imageID)
		}
	}
}

func TestCreatePayloadError(t *testing.T) {
	cfg := config.Config{
		Builder: config.BuilderConfig{
			ID: "testid",
		},
	}
	f, _ := NewFormatter(cfg)

	t.Run("Invalid type", func(t *testing.T) {
		p, err := f.CreatePayload("not a task ref")

		if p != nil {
			t.Errorf("Unexpected payload")
		}
		if err == nil {
			t.Errorf("Expected error")
		} else {
			if err.Error() != "intoto does not support type: not a task ref" {
				t.Errorf("wrong error returned: '%s'", err.Error())
			}
		}
	})

}

func TestCorrectPayloadType(t *testing.T) {
	var i InTotoIte6
	if i.Type() != formats.PayloadTypeInTotoIte6 {
		t.Errorf("Invalid type returned: %s", i.Type())
	}
}
