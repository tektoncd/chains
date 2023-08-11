/*
Copyright 2023 The Tekton Authors
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

package sbom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestGenerateAttestation(t *testing.T) {
	namespace := "test-namespace"
	serviceaccount := "test-serviceaccount"
	tektonObject := objects.NewTaskRunObject(&v1beta1.TaskRun{ //nolint:staticcheck
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec:       v1beta1.TaskRunSpec{ServiceAccountName: serviceaccount},
	})

	cases := []struct {
		name        string
		sbomURL     string
		sbomFormat  string
		imageURL    string
		imageDigest string
		sbom        string
		want        in_toto.Statement
	}{
		{
			name:        "simple",
			sbomURL:     testSBOMRepo,
			sbomFormat:  testSBOMFormat,
			imageURL:    testImageURL,
			imageDigest: testImageDigest,
			sbom:        `{"foo": "bar"}`,
			want: in_toto.Statement{
				StatementHeader: in_toto.StatementHeader{
					Type:          in_toto.StatementInTotoV01,
					PredicateType: testSBOMFormat,
					Subject: []in_toto.Subject{
						{
							Name:   testImageURL,
							Digest: slsa.DigestSet{"sha256": testImageDigestNoAlgo},
						},
					},
				},
				Predicate: json.RawMessage(`{"foo": "bar"}`),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)

			// Setup kubernetes resources
			kc := fakekubeclient.Get(ctx)
			if _, err := kc.CoreV1().ServiceAccounts(namespace).Create(ctx, &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: serviceaccount, Namespace: namespace},
			}, metav1.CreateOptions{}); err != nil {
				t.Fatal(err)
			}

			// Setup registry
			registry := httptest.NewServer(registry.New())
			defer registry.Close()
			registryURL, err := url.Parse(registry.URL)
			if err != nil {
				t.Fatal(err)
			}

			// Populate image references with registry host
			imageURL := fmt.Sprintf(c.imageURL, registryURL.Host)

			// Create SBOM image.
			sbomURL, err := pushSBOMLayer(fmt.Sprintf(c.sbomURL, registryURL.Host), c.sbom)
			if err != nil {
				t.Fatal(err)
			}

			// Setup SBOM Object
			sbomObject := objects.NewSBOMObject(
				sbomURL.String(),
				c.sbomFormat,
				imageURL,
				c.imageDigest, tektonObject)

			var maxSBOMBytes int64 = 10 * 1024 * 1024
			got, err := GenerateAttestation(ctx, kc, testBuilderID, maxSBOMBytes, sbomObject)
			if err != nil {
				t.Fatal(err)
			}

			transformer := cmp.Transformer("registry", func(subject in_toto.Subject) in_toto.Subject {
				if strings.Contains(subject.Name, "%s") {
					subject.Name = fmt.Sprintf(subject.Name, registryURL.Host)
				}
				return subject
			})
			if !cmp.Equal(got, c.want, transformer) {
				t.Errorf("GenerateAttestation() = %s", cmp.Diff(got, c.want, transformer))
			}
		})
	}

}

const (
	testSBOMRepo          = "%s/foo/bat"
	testSBOMFormat        = "https://cyclonedx.org/schema"
	testSBOMMediaType     = "application/octet-stream"
	testImageURL          = "%s/foo/bat:latest"
	testImageDigestNoAlgo = "05f95b26ed10668b7183c1e2da98610e91372fa9f510046d4ce5812addad86b6"
	testImageDigest       = "sha256:" + testImageDigestNoAlgo
	testBuilderID         = "test-builder-id"
)

func pushSBOMLayer(ref string, data string) (name.Digest, error) {
	layer := stream.NewLayer(io.NopCloser(strings.NewReader(data)))

	repo, err := name.NewRepository(ref)
	if err != nil {
		return name.Digest{}, err
	}

	if err := remote.WriteLayer(repo, layer); err != nil {
		return name.Digest{}, err
	}

	digest, err := layer.Digest()
	if err != nil {
		return name.Digest{}, err
	}

	return repo.Digest(digest.String()), nil
}
