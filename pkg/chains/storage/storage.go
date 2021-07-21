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

package storage

import (
	"github.com/tektoncd/chains/pkg/chains/storage/docdb"
	"github.com/tektoncd/chains/pkg/chains/storage/gcs"
	"github.com/tektoncd/chains/pkg/chains/storage/oci"
	"github.com/tektoncd/chains/pkg/chains/storage/tekton"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
)

// Backend is an interface to store a chains Payload
type Backend interface {
	StorePayload(rawPayload []byte, signature string, opts config.StorageOpts) error
	Type() string
}

// InitializeBackends creates and initializes every configured storage backend.
func InitializeBackends(ps versioned.Interface, logger *zap.SugaredLogger, tr *v1beta1.TaskRun, cfg config.Config) (map[string]Backend, error) {
	// Add an entry here for every configured backend
	configuredBackends := []string{
		cfg.Artifacts.TaskRuns.StorageBackend,
		cfg.Artifacts.OCI.StorageBackend}

	// Now only initialize and return the configured ones.
	backends := map[string]Backend{}
	for _, backendType := range configuredBackends {
		switch backendType {
		case gcs.StorageBackendGCS:
			gcsBackend, err := gcs.NewStorageBackend(logger, tr, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = gcsBackend
		case tekton.StorageBackendTekton:
			backends[backendType] = tekton.NewStorageBackend(ps, logger, tr)
		case oci.StorageBackendOCI:
			ociBackend, err := oci.NewStorageBackend(logger, tr, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = ociBackend
		case docdb.StorageTypeDocDB:
			docdbBackend, err := docdb.NewStorageBackend(logger, tr, cfg)
			if err != nil {
				return nil, err
			}
			backends[backendType] = docdbBackend
		}
	}
	return backends, nil
}
