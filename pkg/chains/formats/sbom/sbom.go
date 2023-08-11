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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"k8s.io/client-go/kubernetes"
)

func GenerateAttestation(ctx context.Context, kc kubernetes.Interface, builderID string, maxBytes int64, sbom *objects.SBOMObject) (interface{}, error) {
	subject := []intoto.Subject{
		{Name: sbom.GetImageURL(), Digest: toDigestSet(sbom.GetImageDigest())},
	}

	data, err := getData(ctx, kc, maxBytes, sbom)
	if err != nil {
		return nil, err
	}

	att := intoto.Statement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: sbom.GetSBOMFormat(),
			Subject:       subject,
		},
		Predicate: data,
	}
	return att, nil
}

func getData(ctx context.Context, kc kubernetes.Interface, maxBytes int64, sbom *objects.SBOMObject) (json.RawMessage, error) {
	opt, err := sbom.OCIRemoteOption(ctx, kc)
	if err != nil {
		return nil, err
	}

	uri := sbom.GetSBOMURL()
	ref, err := name.NewDigest(uri)
	if err != nil {
		return nil, err
	}

	rawLayer, err := remote.Layer(ref, opt)
	if err != nil {
		return nil, err
	}

	layer, err := rawLayer.Uncompressed()
	if err != nil {
		return nil, err
	}
	defer layer.Close()

	var blob bytes.Buffer
	if _, err := io.Copy(&blob, io.LimitReader(layer, maxBytes)); err != nil {
		return nil, err
	}

	var data json.RawMessage
	if err := json.Unmarshal(blob.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("SBOM is not valid JSON or is too large: %w", err)
	}
	return data, nil
}

func toDigestSet(digest string) slsa.DigestSet {
	algo, value, found := strings.Cut(digest, ":")
	if !found {
		value = algo
		algo = "sha256"
	}
	return slsa.DigestSet{algo: value}
}
