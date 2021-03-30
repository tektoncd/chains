package formats

import (
	"encoding/json"
	"testing"

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
      }
    ],
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
    "podName": "go-build-pipeline-run-tcjlv-docker-dfhs4-pod-tkkkv",
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
      }
    ],
    "taskSpec": {
      "params": [
        {
          "name": "IMAGE",
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
        }
      ]
    }
  }
}
`

var expected = in_toto.Provenance{
	Attestation: in_toto.Attestation{
		AttestationType: in_toto.ProvenanceTypeV1,
		Subject: in_toto.ArtifactCollection{
			"test.io/test/image": in_toto.ArtifactDigest{
				"sha256": "hash4",
			},
		},
		Materials: in_toto.ArtifactCollection{
			"gcr.io/test1/test1": in_toto.ArtifactDigest{
				"sha256": "hash1",
			},
			"gcr.io/test2/test2": in_toto.ArtifactDigest{
				"sha256": "hash2",
			},
			"gcr.io/test3/test3": in_toto.ArtifactDigest{
				"sha256": "hash3",
			},
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

	if diff := cmp.Diff(got, expected); diff != "" {
		t.Errorf("InTotoIte6.CreatePayload(): -want +got: %s", diff)
	}
}
