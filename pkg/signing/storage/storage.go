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
	"github.com/tektoncd/chains/pkg/signing/storage/tekton"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
)

// Backend is an interface to store a chains Payload
type Backend interface {
	StorePayload(signed []byte, signature string, key string) error
	Type() string
}

// InitializeBackends creates and initializes every configured storage backend.
func InitializeBackends(ps versioned.Interface, logger *zap.SugaredLogger, tr *v1beta1.TaskRun) []Backend {
	backends := []Backend{}

	// Add one entry here for every storage backend type.
	backends = append(backends, tekton.NewStorageBackend(ps, logger, tr))

	return backends
}
