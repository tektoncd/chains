/*
Copyright 2020 The Tekton Authors
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

package oci

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/sigstore/sigstore/pkg/signature/payload"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	remotetest "github.com/tektoncd/pipeline/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	logtesting "knative.dev/pkg/logging/testing"
)

const namespace = "oci-test"

var (
	tr = &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
	}
	pr = &v1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: namespace,
		},
	}
)

func TestBackend_StorePayload(t *testing.T) {
	// Create registry server
	s := httptest.NewServer(registry.New())
	defer s.Close()
	u, _ := url.Parse(s.URL)

	// Push image to server
	ref, err := remotetest.CreateImage(u.Host+"/task/"+tr.Name, tr)
	if err != nil {
		t.Fatalf("failed to push img: %v", err)
	}
	digest := strings.Split(ref, "@")[1]
	digSplits := strings.Split(digest, ":")
	algo, hex := digSplits[0], digSplits[1]

	simple := simple.SimpleContainerImage{
		Critical: payload.Critical{
			Identity: payload.Identity{
				DockerReference: u.Host + "/task/" + tr.Name,
			},
			Image: payload.Image{
				DockerManifestDigest: digest,
			},
			Type: payload.CosignSignatureType,
		},
	}

	intotoStatement := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject: []in_toto.Subject{
				{
					Name: u.Host + "/task/" + tr.Name,
					Digest: common.DigestSet{
						algo: hex,
					},
				},
			},
		},
		Predicate: slsa.ProvenancePredicate{},
	}

	type fields struct {
		object objects.TektonObject
	}
	type args struct {
		payload     interface{}
		signature   string
		storageOpts config.StorageOpts
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{{
		name: "simplesigning payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "simplesigning",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			},
		},
		wantErr: false,
	}, {
		name: "into-to payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   intotoStatement,
			signature: "into-to",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "no subject",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   in_toto.Statement{},
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "simplesigning payload",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   simple,
			signature: "simplesigning",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSimpleSigning,
			},
		},
		wantErr: false,
	}, {
		name: "into-to payload",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   intotoStatement,
			signature: "into-to",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "in-toto-and-simple-payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}, {
		name: "tekton-and-simple-payload",
		fields: fields{
			object: objects.NewTaskRunObjectV1(tr),
		},
		args: args{
			payload:   simple,
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: "tekton",
			},
		},
		wantErr: false,
	}, {
		name: "no subject",
		fields: fields{
			object: objects.NewPipelineRunObjectV1(pr),
		},
		args: args{
			payload:   in_toto.Statement{},
			signature: "",
			storageOpts: config.StorageOpts{
				PayloadFormat: formats.PayloadTypeSlsav1,
			},
		},
		wantErr: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logtesting.TestContextWithLogger(t)
			b := &Backend{
				getAuthenticator: func(context.Context, objects.TektonObject, kubernetes.Interface) (remote.Option, error) {
					return remote.WithAuthFromKeychain(authn.DefaultKeychain), nil
				},
			}
			rawPayload, err := json.Marshal(tt.args.payload)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			if err := b.StorePayload(ctx, tt.fields.object, rawPayload, tt.args.signature, tt.args.storageOpts); (err != nil) != tt.wantErr {
				t.Errorf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
