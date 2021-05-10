package intotoite6

import (
	"encoding/json"
	"testing"

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
        "imageID": "docker-pullable://gcr.io/test1/test1@sha256:hash1"
      },
      {
        "name": "step2",
        "container": "step-step2",
        "imageID": "docker-pullable://gcr.io/test2/test2@sha256:hash2"
      },
      {
        "name": "step3",
        "container": "step-step3",
        "imageID": "docker-pullable://gcr.io/test3/test3@sha256:hash3"
      }
    ],
    "taskResults": [
      {
        "name": "IMAGE-DIGEST",
        "value": "sha256:hash4"
      },
      {
        "name": "filename-DIGEST",
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
          "name": "IMAGE-DIGEST",
          "description": "Digest of the image just built."
        },
        {
          "name": "filename-DIGEST",
          "description": "Digest of the file just built."
        }
      ]
    }
  }
}
`

var expected1 = in_toto.ProvenanceStatement{
	StatementHeader: in_toto.StatementHeader{
		Type:          in_toto.StatementInTotoV01,
		PredicateType: in_toto.PredicateProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "/bin/ls",
				Digest: map[string]string{
					"sha256": "hash5",
				},
			},
			{
				Name: "test.io/test/image",
				Digest: map[string]string{
					"sha256": "hash4",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "gcr.io/test1/test1",
				Digest: map[string]string{
					"sha256": "hash1",
				},
			},
			{
				URI: "gcr.io/test2/test2",
				Digest: map[string]string{
					"sha256": "hash2",
				},
			},
			{
				URI: "gcr.io/test3/test3",
				Digest: map[string]string{
					"sha256": "hash3",
				},
			},
			{
				URI: "git+https://git.test.com",
				Digest: map[string]string{
					"git_commit": "abcd",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test-pod-name",
		},
		Recipe: in_toto.ProvenanceRecipe{
			Type:              tektonID,
			EntryPoint:        "test-task",
			DefinedInMaterial: intP(3),
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
        "imageID": "docker-pullable://gcr.io/test1/test1@sha256:hash1"
      }
    ],
    "taskResults": [
      {
        "name": "some-uri-DIGEST",
        "value": "sha256:hash123"
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
          "name": "some-uri-DIGEST",
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
		PredicateType: in_toto.PredicateProvenanceV01,
		Subject: []in_toto.Subject{
			{
				Name: "pkg:deb/debian/curl@7.50.3-1",
				Digest: map[string]string{
					"sha256": "hash123",
				},
			},
		},
	},
	Predicate: in_toto.ProvenancePredicate{
		Materials: []in_toto.ProvenanceMaterial{
			{
				URI: "gcr.io/test1/test1",
				Digest: map[string]string{
					"sha256": "hash1",
				},
			},
		},
		Builder: in_toto.ProvenanceBuilder{
			ID: "test-pod-name",
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

	i := &InTotoIte6{}
	got, err := i.CreatePayload(&tr)

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

	i := &InTotoIte6{}
	got, err := i.CreatePayload(&tr)

	if diff := cmp.Diff(expected2, got); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}

func TestNewFormater(t *testing.T) {
	t.Run("Ok", func(t *testing.T) {
		cfg := config.Config{
			Builder: config.BuilderConfig{
				ID: "testid",
			},
		}
		f, err := NewFormater(cfg)
		if f == nil {
			t.Error("Failed to create formater")
		}
		if err != nil {
			t.Errorf("Error creating formater: %s", err)
		}
	})
	t.Run("Fail", func(t *testing.T) {
		cfg := config.Config{}
		f, err := NewFormater(cfg)
		if f != nil {
			t.Error("Expected to create to fail")
		}
		if err != ErrNoBuilderID {
			t.Errorf("Unexpected error : %s", err)
		}
	})
}
