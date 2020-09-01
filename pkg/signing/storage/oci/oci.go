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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"path"

	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"knative.dev/pkg/system"
)

const (
	StorageBackendOCI = "oci"
	mediaType         = "application/vnd.tekton.chains.signature.v1+json"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	logger *zap.SugaredLogger
	tr     *v1beta1.TaskRun
	cfg    config.Config
	auth   remote.Option
}

// NewStorageBackend returns a new OCI StorageBackend that stores signatures in an OCI registry
func NewStorageBackend(logger *zap.SugaredLogger, tr *v1beta1.TaskRun, cfg config.Config) (*Backend, error) {
	kc, err := k8schain.NewInCluster(k8schain.Options{Namespace: system.Namespace(), ServiceAccountName: "tekton-pipelines-controller"})
	if err != nil {
		return nil, err
	}

	return &Backend{
		logger: logger,
		tr:     tr,
		cfg:    cfg,
		auth:   remote.WithAuthFromKeychain(kc),
	}, nil
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(signed []byte, signature string, key string) error {
	b.logger.Infof("Storing payload on TaskRun %s/%s", b.tr.Namespace, b.tr.Name)

	img, err := createImage(signed, signature)
	if err != nil {
		return err
	}

	imageRoot := fmt.Sprintf("taskrun-%s-%s-%s", b.tr.Namespace, b.tr.Name, b.tr.UID)
	imageName := path.Join(imageRoot, key)
	ref, err := name.ParseReference(path.Join(b.cfg.Storage.OCI.Repository, imageName), name.WeakValidation)
	if err != nil {
		return err
	}

	b.logger.Infof("Pushing signature image to %s", ref.Name())
	if err := writeImage(ref, img, b.auth); err != nil {
		return err
	}
	return nil
}

// Set as a variable for mocking
var writeImage = remote.Write

func createImage(signed []byte, signature string) (v1.Image, error) {
	buf := bytes.Buffer{}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name: "signed",
		Mode: 0600,
		Size: int64(len(signed)),
	}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(signed); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	bodyLayer, err := tarball.LayerFromReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}

	img, err := mutate.Append(empty.Image, mutate.Addendum{
		Annotations: map[string]string{
			"signature": signature,
		},
		MediaType: mediaType,
		Layer:     bodyLayer,
	})
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (b *Backend) Type() string {
	return StorageBackendOCI
}
