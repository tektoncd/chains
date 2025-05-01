/*
Copyright 2025 The Tekton Authors
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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	archivistaClient "github.com/in-toto/archivista/pkg/http-client"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testProvenanceDSSE = `{"payload":"eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJodHRwczovL3Nsc2EuZGV2L3Byb3ZlbmFuY2UvdjAuMiIsInByZWRpY2F0ZSI6eyJidWlsZENvbmZpZyI6eyJ0YXNrcyI6W3siZmluaXNoZWRPbiI6IjIwMjUtMDItMjdUMTc6MTY6MzFaIiwiaW52b2NhdGlvbiI6eyJjb25maWdTb3VyY2UiOnt9LCJlbnZpcm9ubWVudCI6eyJhbm5vdGF0aW9ucyI6eyJwaXBlbGluZS50ZWt0b24uZGV2L3JlbGVhc2UiOiJjNmQzOGM5In0sImxhYmVscyI6eyJhcHAua3ViZXJuZXRlcy5pby9tYW5hZ2VkLWJ5IjoidGVrdG9uLXBpcGVsaW5lcyIsInRla3Rvbi5kZXYvbWVtYmVyT2YiOiJ0YXNrcyIsInRla3Rvbi5kZXYvcGlwZWxpbmUiOiJoZWxsby13b3JsZC1waXBlbGluZSIsInRla3Rvbi5kZXYvcGlwZWxpbmVSdW4iOiJoZWxsby13b3JsZC1waXBlbGluZS1ydW4tMTc0MDY3NjU3OSIsInRla3Rvbi5kZXYvcGlwZWxpbmVSdW5VSUQiOiIyMDhlMjdmOS0zOWM4LTQxOWEtOGY2MC1kY2UzMmMxNDlhODgiLCJ0ZWt0b24uZGV2L3BpcGVsaW5lVGFzayI6InNheS1oZWxsbyIsInRla3Rvbi5kZXYvdGFzayI6ImhlbGxvLXdvcmxkLXRhc2sifX0sInBhcmFtZXRlcnMiOnt9fSwibmFtZSI6InNheS1oZWxsbyIsInJlZiI6eyJraW5kIjoiVGFzayIsIm5hbWUiOiJoZWxsby13b3JsZC10YXNrIn0sInJlc3VsdHMiOlt7Im5hbWUiOiJQSVBFTElORV9SVU5fQVJUSUZBQ1RfRElHRVNUIiwidHlwZSI6InN0cmluZyIsInZhbHVlIjoic2hhMjU2OjQ3OTJjZTEyMTBmZWRmNWY1MWZjMTRiZDFiZjAyOGFmNzFkMmU0NWE3ZTc0YzRhYmVjZGIzYzc3NGZlNjNmNmYifSx7Im5hbWUiOiJQSVBFTElORV9SVU5fQVJUSUZBQ1RfVVJJIiwidHlwZSI6InN0cmluZyIsInZhbHVlIjoiJChjb250ZXh0LnBpcGVsaW5lUnVuLm5hbWUpIn1dLCJzZXJ2aWNlQWNjb3VudE5hbWUiOiJkZWZhdWx0Iiwic3RhcnRlZE9uIjoiMjAyNS0wMi0yN1QxNzoxNjoxOVoiLCJzdGF0dXMiOiJTdWNjZWVkZWQiLCJzdGVwcyI6W3siYW5ub3RhdGlvbnMiOm51bGwsImFyZ3VtZW50cyI6bnVsbCwiZW50cnlQb2ludCI6IiMhL2Jpbi9iYXNoXG5lY2hvIFwiSGVsbG8gV29ybGRcIlxuZWNobyBcIlBpcGVsaW5lUnVuIG5hbWUgZnJvbSBlbnY6ICRQSVBFTElORV9SVU5fTkFNRVwiXG4jIFdyaXRlIHRoZSBQaXBlbGluZVJ1biBJRCBhcyB0aGUgYXJ0aWZhY3QgVVJJIHJlc3VsdFxuZWNobyAtbiBcIiRQSVBFTElORV9SVU5fTkFNRVwiID4gL3Rla3Rvbi9yZXN1bHRzL1BJUEVMSU5FX1JVTl9BUlRJRkFDVF9VUklcbiMgQ29tcHV0ZSB0aGUgU0hBMjU2IGRpZ2VzdCBvZiB0aGUgUGlwZWxpbmVSdW4gSURcbmRpZ2VzdD0kKGVjaG8gLW4gXCIkUElQRUxJTkVfUlVOX05BTUVcIiB8IHNoYTI1NnN1bSB8IGF3ayAne3ByaW50ICQxfScpXG4jIFdyaXRlIHRoZSBkaWdlc3QgKHByZWZpeGVkIHdpdGggXCJzaGEyNTY6XCIpIGFzIHRoZSBhcnRpZmFjdCBkaWdlc3QgcmVzdWx0XG5lY2hvIC1uIFwic2hhMjU2OiRkaWdlc3RcIiA+IC90ZWt0b24vcmVzdWx0cy9QSVBFTElORV9SVU5fQVJUSUZBQ1RfRElHRVNUXG4iLCJlbnZpcm9ubWVudCI6eyJjb250YWluZXIiOiJwcmludC1oZWxsbyIsImltYWdlIjoib2NpOi8vdWJ1bnR1QHNoYTI1Njo4ZTVjNGYwMjg1ZWNiYjRlYWQwNzA0MzFkMjliNTc2YTUzMGQzMTY2ZGY3M2VjNDRhZmZjMWNkMjc1NTUxNDFiIn19XX1dfSwiYnVpbGRUeXBlIjoidGVrdG9uLmRldi92MWJldGExL1BpcGVsaW5lUnVuIiwiYnVpbGRlciI6eyJpZCI6Imh0dHBzOi8vdGVrdG9uLmRldi9jaGFpbnMvdjIifSwiaW52b2NhdGlvbiI6eyJjb25maWdTb3VyY2UiOnt9LCJlbnZpcm9ubWVudCI6eyJsYWJlbHMiOnsidGVrdG9uLmRldi9waXBlbGluZSI6ImhlbGxvLXdvcmxkLXBpcGVsaW5lIn19LCJwYXJhbWV0ZXJzIjp7fX0sIm1hdGVyaWFscyI6W3siZGlnZXN0Ijp7InNoYTI1NiI6IjhlNWM0ZjAyODVlY2JiNGVhZDA3MDQzMWQyOWI1NzZhNTMwZDMxNjZkZjczZWM0NGFmZmMxY2QyNzU1NTE0MWIifSwidXJpIjoib2NpOi8vdWJ1bnR1In1dLCJtZXRhZGF0YSI6eyJidWlsZEZpbmlzaGVkT24iOiIyMDI1LTAyLTI3VDE3OjE2OjMxWiIsImJ1aWxkU3RhcnRlZE9uIjoiMjAyNS0wMi0yN1QxNzoxNjoxOVoiLCJjb21wbGV0ZW5lc3MiOnsiZW52aXJvbm1lbnQiOmZhbHNlLCJtYXRlcmlhbHMiOmZhbHNlLCJwYXJhbWV0ZXJzIjpmYWxzZX0sInJlcHJvZHVjaWJsZSI6ZmFsc2V9fX0=","payloadType":"application/vnd.in-toto+json","signatures":[{"keyid":"SHA256:ZnwkOhDkkbPcS5pY0SqpimYAJl2pqgHrxW9ECLcZvJ8","sig":"MEQCIHfE2iwOj13IJLoMXCQ2VvdkOvccX2BnaGZSr/m6+WPCAiAyK1HpCiHNBHJvyPJDl7cQIHtNkJQxLBGUDUsnycpfzQ=="}]}`

// --------------------------
// Helper: setupEnv
// --------------------------

// setupEnv creates a fresh ArchivistaStorage test environment using a given TaskRun name.
func setupEnv(taskRunName string, cfg config.Config, archClient *archivistaClient.ArchivistaClient) (*Backend, *objects.TaskRunObjectV1) {
	tr := &v1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskRunName,
			Namespace: "default",
		},
	}
	obj := objects.NewTaskRunObjectV1(tr)
	aStorage, err := NewStorageBackend(cfg)
	if err != nil {
		panic("failed to initialize ArchivistaStorage: " + err.Error())
	}
	// Override the Archivista client with the provided one.
	aStorage.client = archClient
	return aStorage, obj
}

// --------------------------
// StorePayload Tests
// --------------------------

// TestStorePayload_TaskRun tests the basic success path of StorePayload without certificate data.
func TestStorePayload_TaskRun(t *testing.T) {
	ctx := context.Background()

	// Set up an httptest server to simulate Archivista.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upload" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"gitoid": "fake-gitoid"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	httpClient := &http.Client{}
	cfg := config.Config{
		Storage: config.StorageConfigs{
			Archivista: config.ArchivistaStorageConfig{
				URL: ts.URL,
			},
		},
	}
	archClient, err := archivistaClient.CreateArchivistaClient(httpClient, cfg.Storage.Archivista.URL)
	if err != nil {
		t.Fatalf("failed to create Archivista client: %v", err)
	}

	aStorage, obj := setupEnv("test-taskrun", cfg, archClient)

	// Prepare a valid payload.
	type mockPayload struct {
		A string `json:"a"`
		B int    `json:"b"`
	}
	payload := mockPayload{
		A: "foo",
		B: 3,
	}
	payloadBytes, err := json.Marshal(payload)
	assert.NoError(t, err, "should marshal payload")
	encodedPayload := base64.StdEncoding.EncodeToString(payloadBytes)
	opts := config.StorageOpts{
		ShortKey: "mockpayload",
		Cert:     "",
		Chain:    "",
	}

	// Call StorePayload.
	err = aStorage.StorePayload(ctx, obj, []byte(encodedPayload), testProvenanceDSSE, opts)
	assert.NoError(t, err, "StorePayload should succeed")
}

// TestStorePayload_ErrorCases exercises error branches in StorePayload.
func TestStorePayload_ErrorCases(t *testing.T) {
	ctx := context.Background()

	// Setup a common httptest server and configuration.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"gitoid": "fake-gitoid"}`))
	}))
	defer ts.Close()

	httpClient := &http.Client{}
	cfg := config.Config{
		Storage: config.StorageConfigs{
			Archivista: config.ArchivistaStorageConfig{
				URL: ts.URL,
			},
		},
	}
	archClient, err := archivistaClient.CreateArchivistaClient(httpClient, cfg.Storage.Archivista.URL)
	if err != nil {
		t.Fatalf("failed to create Archivista client: %v", err)
	}

	t.Run("non-dsse signature", func(t *testing.T) {
		aStorage, obj := setupEnv("non-dsse signature", cfg, archClient)
		err := aStorage.StorePayload(ctx, obj, []byte("dummy"), "abcde12354", config.StorageOpts{})
		assert.ErrorContains(t, err, "Failed to parse DSSE")
	})
}
