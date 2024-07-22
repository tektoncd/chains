/*
Copyright 2021 The Tekton Authors
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

package docdb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"gocloud.dev/docstore"
	_ "gocloud.dev/docstore/memdocstore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestBackend_StorePayload(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)
	type args struct {
		rawPayload interface{}
		signature  string
		key        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no error",
			args: args{
				rawPayload: &v1beta1.TaskRun{ObjectMeta: metav1.ObjectMeta{UID: "foo"}}, //nolint:staticcheck
				signature:  "signature",
				key:        "foo",
			},
		},
		{
			name: "no error - PipelineRun",
			args: args{
				rawPayload: &v1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{UID: "foo"}}, //nolint:staticcheck
				signature:  "signature",
				key:        "moo",
			},
		},
	}

	memURL := "mem://chains/name"
	coll, err := docstore.OpenCollection(ctx, memURL)
	if err != nil {
		t.Fatal(err)
	}
	defer coll.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.WithLogger(ctx, logtesting.TestLogger(t))
			// Prepare the document.
			b := &Backend{
				coll: coll,
			}
			sb, err := json.Marshal(tt.args.rawPayload)
			if err != nil {
				t.Fatal(err)
			}

			// Store the document.
			opts := config.StorageOpts{ShortKey: tt.args.key}
			tektonObj, err := objects.NewTektonObject(tt.args.rawPayload)
			if err != nil {
				t.Fatal(err)
			}
			if err := b.StorePayload(ctx, tektonObj, sb, tt.args.signature, opts); (err != nil) != tt.wantErr {
				t.Fatalf("Backend.StorePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
			obj := SignedDocument{
				Name: tt.args.key,
			}
			if err := coll.Get(ctx, &obj); err != nil {
				t.Fatal(err)
			}

			// Check the signature.
			signatures, err := b.RetrieveSignatures(ctx, tektonObj, opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(signatures[obj.Name]) != 1 {
				t.Fatalf("unexpected number of signatures: %d", len(signatures[obj.Name]))
			}

			if signatures[obj.Name][0] != tt.args.signature {
				t.Errorf("wrong signature, expected %s, got %s", tt.args.signature, signatures[obj.Name][0])
			}

			// Check the payload.
			payloads, err := b.RetrievePayloads(ctx, tektonObj, opts)
			if err != nil {
				t.Fatal(err)
			}
			if payloads[obj.Name] != string(sb) {
				t.Errorf("wrong payload, expected %s, got %s", tt.args.rawPayload, payloads[obj.Name])
			}
		})
	}
}

func TestPopulateMongoServerURL(t *testing.T) {
	mongoDir := t.TempDir()
	mongoEnvFromFile := "mongoEnvFromFile"
	if err := os.WriteFile(filepath.Join(mongoDir, "MONGO_SERVER_URL"), []byte(mongoEnvFromFile), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		cfg              config.Config
		setMongoEnv      string
		expectedMongoEnv string
		wantErr          bool
	}{
		{
			name: "fail when MONGO_SERVER_URL is not set but storage.docdb.url is set",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL: "mongo://chainsdb/chainscollection?id_field=name",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "pass when MONGO_SERVER_URL is set and storage.docdb.url is set",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL: "mongo://chainsdb/chainscollection?id_field=name",
					},
				},
			},
			setMongoEnv:      "testEnv",
			expectedMongoEnv: "testEnv",
			wantErr:          false,
		},
		{
			name: "storage.docdb.mongo-server-url has more precedence than MONGO_SERVER_URL",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL:            "mongo://chainsdb/chainscollection?id_field=name",
						MongoServerURL: "envFromConfig",
					},
				},
			},
			setMongoEnv:      "testEnv",
			expectedMongoEnv: "envFromConfig",
			wantErr:          false,
		},
		{
			name: "storage.docdb.mongo-server-url works solo",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL:            "mongo://chainsdb/chainscollection?id_field=name",
						MongoServerURL: "envFromConfigSolo",
					},
				},
			},
			setMongoEnv:      "",
			expectedMongoEnv: "envFromConfigSolo",
			wantErr:          false,
		},
		{
			name: "storage.docdb.mongo-server-url-dir has precedence over storage.docdb.mongo-server-url and MONGO_SERVER_URL",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL:               "mongo://chainsdb/chainscollection?id_field=name",
						MongoServerURLDir: mongoDir,
						MongoServerURL:    "envFromConfig",
					},
				},
			},
			setMongoEnv:      "mongoEnvVar",
			expectedMongoEnv: mongoEnvFromFile,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv("MONGO_SERVER_URL")
			ctx, _ := rtesting.SetupFakeContext(t)

			if tt.setMongoEnv != "" {
				if err := os.Setenv("MONGO_SERVER_URL", tt.setMongoEnv); err != nil {
					t.Error(err)
				}
			}

			if err := populateMongoServerURL(ctx, tt.cfg); (err != nil) != tt.wantErr {
				t.Errorf("did not expect an error, but got: %v", err)
			}

			currentMongoEnv := os.Getenv("MONGO_SERVER_URL")
			if os.Getenv("MONGO_SERVER_URL") != tt.expectedMongoEnv {
				t.Errorf("expected MONGO_SERVER_URL to be: %s, but got: %s", tt.expectedMongoEnv, currentMongoEnv)
			}
		})
	}
}
func TestSetMongoServerURLFromDir(t *testing.T) {
	mongoDir := t.TempDir()
	mongoEnvFromFile := "mongoEnvFromFile"
	if err := os.WriteFile(filepath.Join(mongoDir, "MONGO_SERVER_URL"), []byte(mongoEnvFromFile), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(mongoDir, "just-a-file"), []byte("just-a-file"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name             string
		directory        string
		expectedMongoEnv string
		wantErr          bool
	}{
		{
			name:      "error if path is not a directory",
			directory: filepath.Join(mongoDir, "just-a-file"),
			wantErr:   true,
		},
		{
			name:             "verify if MONGO_SERVER_URL is being set from path",
			directory:        mongoDir,
			expectedMongoEnv: mongoEnvFromFile,
			wantErr:          false,
		},
		{
			name:      "no error if path does not exist (it will be created)",
			directory: filepath.Join(mongoDir, "does-not-exist"),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv("MONGO_SERVER_URL")
			if err := setMongoServerURLFromDir(tt.directory); (err != nil) != tt.wantErr {
				t.Errorf("did not expect an error, but got: %v", err)
			}

			currentEnv := os.Getenv("MONGO_SERVER_URL")
			if currentEnv != tt.expectedMongoEnv {
				t.Errorf("expected MONGO_SERVER_URL: %s, got %s", tt.expectedMongoEnv, currentEnv)
			}
		})
	}
}

func TestWatchBackend(t *testing.T) {
	testEnv := "mongodb://testEnv"

	tests := []struct {
		name             string
		cfg              config.Config
		expectedMongoEnv string
		wantErr          bool
	}{
		{
			name: "ErrNothingToWatch when it's not a MongoDB URL",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL:               "firestore://chainsdb/chainscollection?id_field=name",
						MongoServerURLDir: t.TempDir(),
					},
				},
			},
			expectedMongoEnv: testEnv,
			wantErr:          true,
		},
		{
			name: "ErrNothingToWatch when not storage.docdb.mongo-server-url-dir not set",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL: "mongo://chainsdb/chainscollection?id_field=name",
					},
				},
			},
			expectedMongoEnv: testEnv,
			wantErr:          true,
		},
		{
			name: "verify mongo-server-url-dir/MONGO_SERVER_URL is watched",
			cfg: config.Config{
				Storage: config.StorageConfigs{
					DocDB: config.DocDBStorageConfig{
						URL:               "mongo://chainsdb/chainscollection?id_field=name",
						MongoServerURLDir: t.TempDir(),
					},
				},
			},
			expectedMongoEnv: "mongodb://updatedEnv",
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)

			if err := os.Setenv("MONGO_SERVER_URL", testEnv); err != nil {
				t.Fatal(err)
			}

			watcherStop := make(chan bool)
			defer func() {
				select {
				case watcherStop <- true:
					t.Log("sent close event to fsnotify")
				default:
					t.Log("could not send close event to fsnotify")
				}
			}()

			backendChan, err := WatchBackend(ctx, tt.cfg, watcherStop)
			if (err != nil) != tt.wantErr {
				t.Errorf("did not expect an error, but got: %v", err)
			}

			if tt.wantErr {
				return
			}

			currentEnv := os.Getenv("MONGO_SERVER_URL")
			if currentEnv != testEnv {
				t.Errorf("expected MONGO_SERVER_URL: %s, but got %s", testEnv, currentEnv)
			}

			// Updating file now
			if err := os.WriteFile(filepath.Join(tt.cfg.Storage.DocDB.MongoServerURLDir, "MONGO_SERVER_URL"), []byte(tt.expectedMongoEnv), 0644); err != nil {
				t.Error(err)
			}

			// Let's wait for the event to be read by fsnotify
			time.Sleep(500 * time.Millisecond)

			// Empty the channel now
			<-backendChan
			currentEnv = os.Getenv("MONGO_SERVER_URL")
			if currentEnv != tt.expectedMongoEnv {
				t.Errorf("expected MONGO_SERVER_URL: %s, but got %s", tt.expectedMongoEnv, currentEnv)
			}

			// Let's go back to older env (env rotation) and test again
			if err := os.WriteFile(filepath.Join(tt.cfg.Storage.DocDB.MongoServerURLDir, "MONGO_SERVER_URL"), []byte(testEnv), 0644); err != nil {
				t.Error(err)
			}

			// Let's wait for the event to be read by fsnotify
			time.Sleep(500 * time.Millisecond)

			currentEnv = os.Getenv("MONGO_SERVER_URL")
			if currentEnv != testEnv {
				t.Errorf("expected MONGO_SERVER_URL: %s, but got %s", testEnv, currentEnv)
			}
		})
	}
}
