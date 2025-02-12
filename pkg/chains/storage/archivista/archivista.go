// archivista.go
package archivista

import (
	"context"
	"fmt"

	"github.com/in-toto/archivista/pkg/api"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"
)

const (
	StorageBackendArchivista = "archivista"
)

// Backend is the interface that all storage backends must implement.
// (This is the interface used by the Chains storage initializer.)
type Backend interface {
	StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error
	RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error)
	RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error)
	Type() string
}

// ArchivistaClient defines the subset of methods used from the Archivista API client.
type ArchivistaClient interface {
	Upload(ctx context.Context, req *api.UploadRequest) (*api.UploadResponse, error)
	GetArtifact(ctx context.Context, key string) (*api.Artifact, error)
}

// ArchivistaStorage implements the Backend interface for Archivista.
type ArchivistaStorage struct {
	client ArchivistaClient
	url    string
	cfg    config.ArchivistaStorageConfig
}

// NewArchivistaStorage initializes a new ArchivistaStorage backend.
// It extracts the Archivista-specific configuration from the top-level config.
func NewArchivistaStorage(cfg config.Config) (*ArchivistaStorage, error) {
	archCfg := cfg.Storage.Archivista
	if archCfg.URL == "" {
		return nil, fmt.Errorf("missing archivista URL in storage configuration")
	}

	client, err := api.NewClient(archCfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to create archivista client: %w", err)
	}

	return &ArchivistaStorage{
		client: client,
		url:    archCfg.URL,
		cfg:    archCfg,
	}, nil
}

// StorePayload uploads the payload and signature to Archivista.
// It expects opts.ShortKey (a string) to be set and uses that as the key.
func (a *ArchivistaStorage) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	key := opts.ShortKey
	if key == "" {
		return fmt.Errorf("missing key in storage options (opts.ShortKey)")
	}

	uploadReq := &api.UploadRequest{
		Payload:   rawPayload,
		Signature: []byte(signature),
		KeyID:     key,
	}

	logger.Infof("Uploading payload to Archivista for key %q", key)
	uploadResp, err := a.client.Upload(ctx, uploadReq)
	if err != nil {
		logger.Errorw("Failed to upload payload to Archivista", "key", key, "error", err)
		return fmt.Errorf("failed to upload to archivista: %w", err)
	}

	logger.Infof("Successfully uploaded to Archivista")
	_ = uploadResp // suppress unused variable warning

	return nil
}

// RetrievePayload is our internal method that retrieves a payload and signature from Archivista using the key.
func (a *ArchivistaStorage) RetrievePayload(ctx context.Context, key string) ([]byte, []byte, error) {
	logger := logging.FromContext(ctx)
	logger.Infof("Retrieving artifact from Archivista for key %q", key)
	artifact, err := a.client.GetArtifact(ctx, key)
	if err != nil {
		logger.Errorw("Failed to retrieve artifact from Archivista", "key", key, "error", err)
		return nil, nil, fmt.Errorf("failed to retrieve artifact from archivista: %w", err)
	}
	logger.Infof("Successfully retrieved artifact for key %q", key)
	return artifact.Payload, artifact.Signature, nil
}

// RetrievePayloads implements the Backend interface.
// It calls the internal RetrievePayload and returns a map from key to payload (as a string).
func (a *ArchivistaStorage) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	key := opts.ShortKey
	payload, _, err := a.RetrievePayload(ctx, key)
	if err != nil {
		return nil, err
	}
	return map[string]string{key: string(payload)}, nil
}

// RetrieveSignatures implements the Backend interface.
// It calls the internal RetrievePayload and returns a map from key to a slice containing the signature.
func (a *ArchivistaStorage) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	key := opts.ShortKey
	_, signature, err := a.RetrievePayload(ctx, key)
	if err != nil {
		return nil, err
	}
	return map[string][]string{key: {string(signature)}}, nil
}

// Type returns the storage backend type.
func (a *ArchivistaStorage) Type() string {
	return StorageBackendArchivista
}
