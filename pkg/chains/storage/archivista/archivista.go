// archivista.go
package archivista

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/in-toto/go-witness/dsse"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"knative.dev/pkg/logging"
)

const (
	StorageBackendArchivista = "archivista"
)

// --------------------------------------------------------------------------
// API Types (these replace the types that were previously defined in client.go)
// --------------------------------------------------------------------------

// UploadRequest is the payload that Archivista expects when uploading an artifact.
type UploadRequest struct {
	ArtifactType string `json:"artifactType"`
	Payload      []byte `json:"payload"`
	Signature    []byte `json:"signature"`
	KeyID        string `json:"keyID"`
}

// UploadResponse represents the response returned by Archivista after a successful upload.
type UploadResponse struct {
	Gitoid string `json:"gitoid"`
}

// Artifact represents an artifact downloaded from Archivista.
type Artifact struct {
	Payload   []byte `json:"payload"`
	Signature []byte `json:"signature"`
}

// --------------------------------------------------------------------------
// ArchivistaClient Interface and HTTP Wrapper Implementation
// --------------------------------------------------------------------------

// ArchivistaClient defines the subset of methods we need to call the Archivista API.
type ArchivistaClient interface {
	Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error)
	GetArtifact(ctx context.Context, key string) (*Artifact, error)
}

// httpArchivistaClient is a thin HTTP wrapper that implements ArchivistaClient.
type httpArchivistaClient struct {
	baseURL    string
	httpClient *http.Client
}

// Upload sends the given UploadRequest to Archivista's /upload endpoint.
func (c *httpArchivistaClient) Upload(ctx context.Context, req *UploadRequest) (*UploadResponse, error) {
	uploadURL, err := url.JoinPath(c.baseURL, "upload")
	if err != nil {
		return nil, err
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(respBytes))
	}

	var uploadResp UploadResponse
	if err := json.Unmarshal(respBytes, &uploadResp); err != nil {
		return nil, err
	}
	return &uploadResp, nil
}

// GetArtifact retrieves an artifact from Archivista by key using the /download endpoint.
func (c *httpArchivistaClient) GetArtifact(ctx context.Context, key string) (*Artifact, error) {
	downloadURL, err := url.JoinPath(c.baseURL, "download", key)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(respBytes))
	}

	var artifact Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

// --------------------------------------------------------------------------
// Helper Functions
// --------------------------------------------------------------------------

// dsseEnvelopeToUploadRequest converts a DSSE envelope into an UploadRequest.
// It uses the envelope's PayloadType as the ArtifactType and the first signature.
func dsseEnvelopeToUploadRequest(env dsse.Envelope) *UploadRequest {
	var sig dsse.Signature
	if len(env.Signatures) > 0 {
		sig = env.Signatures[0]
	}
	return &UploadRequest{
		ArtifactType: env.PayloadType,
		Payload:      env.Payload,
		Signature:    sig.Signature,
		KeyID:        sig.KeyID,
	}
}

// getKey extracts the key from the storage options.
func getKey(opts config.StorageOpts) (string, error) {
	if opts.ShortKey == "" {
		return "", fmt.Errorf("missing key in storage options (opts.ShortKey)")
	}
	return opts.ShortKey, nil
}

// buildEnvelope constructs a DSSE envelope from the raw payload, signature, and key.
func buildEnvelope(rawPayload []byte, signature, key string) dsse.Envelope {
	return dsse.Envelope{
		Payload:     rawPayload,
		PayloadType: "tekton-chains",
		Signatures: []dsse.Signature{
			{
				KeyID:     key,
				Signature: []byte(signature),
			},
		},
	}
}

// --------------------------------------------------------------------------
// ArchivistaStorage Implementation (Tekton Backend)
// --------------------------------------------------------------------------

// Backend is the interface that all storage backends must implement.
type Backend interface {
	StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error
	RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error)
	RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error)
	Type() string
}

// ArchivistaStorage implements the Backend interface for Archivista.
type ArchivistaStorage struct {
	client ArchivistaClient
	url    string
	cfg    config.ArchivistaStorageConfig
}

// NewArchivistaStorage initializes a new ArchivistaStorage backend by reading the Archivista URL
// from the top-level config and creating an HTTP client wrapper.
func NewArchivistaStorage(cfg config.Config) (*ArchivistaStorage, error) {
	archCfg := cfg.Storage.Archivista
	if archCfg.URL == "" {
		return nil, fmt.Errorf("missing archivista URL in storage configuration")
	}

	client := &httpArchivistaClient{
		baseURL:    archCfg.URL,
		httpClient: &http.Client{},
	}

	return &ArchivistaStorage{
		client: client,
		url:    archCfg.URL,
		cfg:    archCfg,
	}, nil
}

// StorePayload builds a DSSE envelope from the raw payload and signature,
// converts it to an UploadRequest, and uploads it via the underlying HTTP client.
func (a *ArchivistaStorage) StorePayload(ctx context.Context, obj objects.TektonObject, rawPayload []byte, signature string, opts config.StorageOpts) error {
	logger := logging.FromContext(ctx)
	key, err := getKey(opts)
	if err != nil {
		return err
	}

	env := buildEnvelope(rawPayload, signature, key)
	uploadReq := dsseEnvelopeToUploadRequest(env)

	logger.Infof("Uploading DSSE envelope to Archivista for key %q", key)
	uploadResp, err := a.client.Upload(ctx, uploadReq)
	if err != nil {
		logger.Errorw("Failed to upload DSSE envelope to Archivista", "key", key, "error", err)
		return fmt.Errorf("failed to upload to archivista: %w", err)
	}
	logger.Infof("Successfully uploaded DSSE envelope to Archivista, response: %+v", uploadResp)
	return nil
}

// RetrievePayload is an internal helper that retrieves an artifact from Archivista.
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
// It returns a map from the key to the artifact payload as a string.
func (a *ArchivistaStorage) RetrievePayloads(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string]string, error) {
	key, err := getKey(opts)
	if err != nil {
		return nil, err
	}

	payload, _, err := a.RetrievePayload(ctx, key)
	if err != nil {
		return nil, err
	}
	return map[string]string{key: string(payload)}, nil
}

// RetrieveSignatures implements the Backend interface.
// It returns a map from the key to a slice containing the artifact signature as a string.
func (a *ArchivistaStorage) RetrieveSignatures(ctx context.Context, obj objects.TektonObject, opts config.StorageOpts) (map[string][]string, error) {
	key, err := getKey(opts)
	if err != nil {
		return nil, err
	}

	_, sig, err := a.RetrievePayload(ctx, key)
	if err != nil {
		return nil, err
	}
	return map[string][]string{key: {string(sig)}}, nil
}

// Type returns the storage backend type.
func (a *ArchivistaStorage) Type() string {
	return StorageBackendArchivista
}
