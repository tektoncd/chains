// archivista_test.go
package archivista

import (
	"context"
	"reflect"
	"testing"

	"github.com/in-toto/archivista/pkg/api"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"

	// Import Tekton API types and metav1.
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeArchivistaClient is a fake implementation of ArchivistaClient for testing.
type fakeArchivistaClient struct {
	uploadReq  *api.UploadRequest
	uploadResp *api.UploadResponse
	uploadErr  error

	artifact *api.Artifact
	getErr   error
}

func (f *fakeArchivistaClient) Upload(ctx context.Context, req *api.UploadRequest) (*api.UploadResponse, error) {
	f.uploadReq = req
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	if f.uploadResp != nil {
		return f.uploadResp, nil
	}
	// Return a default response.
	return &api.UploadResponse{}, nil
}

func (f *fakeArchivistaClient) GetArtifact(ctx context.Context, key string) (*api.Artifact, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.artifact, nil
}

// TestStorePayload uses a real Tekton object (a TaskRun converted via the chains objects constructor)
// to exercise StorePayload. We then check that the fake client was called with the expected values.
func TestStorePayload(t *testing.T) {
	ctx := context.Background()

	// Create a fake Archivista client.
	fakeClient := &fakeArchivistaClient{}

	// Build a configuration that includes an Archivista URL.
	cfg := config.Config{
		Storage: config.StorageConfigs{
			Archivista: config.ArchivistaStorageConfig{
				URL: "http://fake.archivista",
			},
		},
	}

	archStorage, err := NewArchivistaStorage(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating ArchivistaStorage: %v", err)
	}
	// Inject our fake client.
	archStorage.client = fakeClient

	// Create a real TaskRun and convert it to a TektonObject.
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-taskrun",
			Namespace: "default",
			UID:       "uid-test",
		},
		// For this test we do not need to set status.
	}
	tektonObj := objects.NewTaskRunObjectV1Beta1(taskRun)

	rawPayload := []byte("test payload")
	signature := "test signature"
	opts := config.StorageOpts{
		ShortKey: "test-key",
		// You can add other fields if needed.
	}

	// Call StorePayload.
	if err := archStorage.StorePayload(ctx, tektonObj, rawPayload, signature, opts); err != nil {
		t.Fatalf("StorePayload returned error: %v", err)
	}

	// Verify that the fake client's Upload method was called with the expected values.
	if fakeClient.uploadReq == nil {
		t.Fatal("Upload was not called on the fake client")
	}
	if string(fakeClient.uploadReq.Payload) != "test payload" {
		t.Errorf("expected payload %q, got %q", "test payload", string(fakeClient.uploadReq.Payload))
	}
	if string(fakeClient.uploadReq.Signature) != "test signature" {
		t.Errorf("expected signature %q, got %q", "test signature", string(fakeClient.uploadReq.Signature))
	}
	if fakeClient.uploadReq.KeyID != "test-key" {
		t.Errorf("expected keyID %q, got %q", "test-key", fakeClient.uploadReq.KeyID)
	}
}

// TestRetrievePayloadsAndSignatures verifies that when the fake client returns a preset artifact,
// the RetrievePayloads and RetrieveSignatures methods return the expected maps.
func TestRetrievePayloadsAndSignatures(t *testing.T) {
	ctx := context.Background()

	// Create a fake client with a preset artifact.
	fakeClient := &fakeArchivistaClient{
		artifact: &api.Artifact{
			Payload:   []byte("retrieved payload"),
			Signature: []byte("retrieved signature"),
		},
	}
	cfg := config.Config{
		Storage: config.StorageConfigs{
			Archivista: config.ArchivistaStorageConfig{
				URL: "http://fake.archivista",
			},
		},
	}
	archStorage, err := NewArchivistaStorage(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating ArchivistaStorage: %v", err)
	}
	archStorage.client = fakeClient

	opts := config.StorageOpts{
		ShortKey: "test-key",
	}
	// Use a real Tekton object.
	taskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-taskrun",
			Namespace: "default",
			UID:       "uid-test",
		},
	}
	tektonObj := objects.NewTaskRunObjectV1Beta1(taskRun)

	// Test RetrievePayloads.
	payloads, err := archStorage.RetrievePayloads(ctx, tektonObj, opts)
	if err != nil {
		t.Fatalf("RetrievePayloads returned error: %v", err)
	}
	expectedPayloads := map[string]string{"test-key": "retrieved payload"}
	if !reflect.DeepEqual(payloads, expectedPayloads) {
		t.Errorf("RetrievePayloads expected %v, got %v", expectedPayloads, payloads)
	}

	// Test RetrieveSignatures.
	sigs, err := archStorage.RetrieveSignatures(ctx, tektonObj, opts)
	if err != nil {
		t.Fatalf("RetrieveSignatures returned error: %v", err)
	}
	expectedSigs := map[string][]string{"test-key": {"retrieved signature"}}
	if !reflect.DeepEqual(sigs, expectedSigs) {
		t.Errorf("RetrieveSignatures expected %v, got %v", expectedSigs, sigs)
	}
}

// TestType verifies that the Type method returns the correct backend type.
func TestType(t *testing.T) {
	cfg := config.Config{
		Storage: config.StorageConfigs{
			Archivista: config.ArchivistaStorageConfig{
				URL: "http://fake.archivista",
			},
		},
	}
	archStorage, err := NewArchivistaStorage(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating ArchivistaStorage: %v", err)
	}
	if archStorage.Type() != StorageBackendArchivista {
		t.Errorf("Type() expected %q, got %q", StorageBackendArchivista, archStorage.Type())
	}
}
