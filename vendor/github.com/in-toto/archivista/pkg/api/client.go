// client.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/in-toto/go-witness/dsse"
)

// Client wraps HTTP calls to an Archivista service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Archivista API client.
func NewClient(baseURL string) (*Client, error) {
	// Validate baseURL.
	_, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}, nil
}

// UploadResponse represents the response from Archivista after a successful upload.

// UploadDSSE uploads a DSSE envelope to Archivista.
// Note that this method now accepts a dsse.Envelope rather than a pointer to an UploadRequest.
func (c *Client) UploadDSSE(ctx context.Context, envelope dsse.Envelope) (*UploadResponse, error) {
	uploadURL, err := url.JoinPath(c.baseURL, "upload")
	if err != nil {
		return nil, err
	}

	bodyBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

// Artifact represents a retrieved artifact from Archivista.
type Artifact struct {
	Payload   []byte `json:"payload"`
	Signature []byte `json:"signature"`
}

// GetArtifact retrieves a DSSE envelope by key from Archivista,
// decodes it as a dsse.Envelope, and converts it into an Artifact.
// It uses the envelope's payload and (if available) the first signature.
func (c *Client) GetArtifact(ctx context.Context, key string) (*Artifact, error) {
	downloadURL, err := url.JoinPath(c.baseURL, "download", key)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.New(string(respBytes))
	}

	// Decode the response into a DSSE envelope.
	var envelope dsse.Envelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}

	// Ensure that at least one signature exists.
	if len(envelope.Signatures) == 0 {
		return nil, errors.New("no signatures in DSSE envelope")
	}

	// Create an Artifact using the envelope's payload and the first signature.
	artifact := &Artifact{
		Payload:   envelope.Payload,
		Signature: envelope.Signatures[0].Signature,
	}
	return artifact, nil
}
