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

package archivista

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/config"

	"github.com/in-toto/archivista/pkg/api"
)

const (
	StorageBackendArchivista = "archivista"
)

// ArchivistaStorage implements the Storage interface for Archivista.
type ArchivistaStorage struct {
	client *api.Client
	url    string
	cfg    config.StorageOpts
}

// NewArchivistaStorage initializes a new ArchivistaStorage.
func NewArchivistaStorage(cfg config.StorageOpts) (*ArchivistaStorage, error) {
	url, ok := cfg["archivista-url"]
	if !ok {
		return nil, fmt.Errorf("missing archivista-url in storage configuration")
	}

	client, err := api.NewClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create archivista client: %w", err)
	}

	return &ArchivistaStorage{
		client: client,
		url:    url,
		cfg:    cfg,
	}, nil
}

// StorePayload uploads the payload and signature to Archivista.
func (a *ArchivistaStorage) StorePayload(ctx context.Context, payload, signature []byte, key string, metadata config.Metadata) error {
	artifactType := metadata["artifactType"]
	if artifactType == "" {
		artifactType = "unknown"
	}

	uploadReq := &api.UploadRequest{
		ArtifactType: artifactType,
		Payload:      payload,
		Signature:    signature,
		KeyID:        key,
	}

	uploadResp, err := a.client.Upload(ctx, uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload to archivista: %w", err)
	}

	fmt.Printf("Successfully uploaded to Archivista: %s\n", uploadResp.ID)
	return nil
}

// Type returns the storage type name.
func (a *ArchivistaStorage) Type() string {
	return "archivista"
}

// RetrievePayload retrieves a payload from Archivista based on the key.
func (a *ArchivistaStorage) RetrievePayload(ctx context.Context, key string) ([]byte, []byte, error) {
	artifact, err := a.client.GetArtifact(ctx, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve artifact from archivista: %w", err)
	}

	return artifact.Payload, artifact.Signature, nil
}

// Ensure this type implements the Storage interface.
var _ Storage = (*ArchivistaStorage)(nil)

// RegisterArchivistaStorage registers the Archivista storage backend.
func RegisterArchivistaStorage() {
	RegisterStorageBackend("archivista", func(cfg config.StorageOpts) (Storage, error) {
		return NewArchivistaStorage(cfg)
	})
}
