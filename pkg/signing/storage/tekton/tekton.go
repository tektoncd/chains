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

package tekton

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
)

const (
	StorageBackendTekton = "tekton"
	PayloadAnnotation    = "chains.tekton.dev/%s-payload"
)

// Tekton is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	pipelienclientset versioned.Interface
	logger            *zap.SugaredLogger
}

// NewStorageBackend returns a new Tekton StorageBackend
func NewStorageBackend(ps versioned.Interface, logger *zap.SugaredLogger) *Backend {
	return &Backend{
		pipelienclientset: ps,
		logger:            logger,
	}
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(payload interface{}, payloadType string, tr *v1beta1.TaskRun) error {
	b.logger.Infof("Storing payload type %s on TaskRun %s/%s", payloadType, tr.Namespace, tr.Name)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	textPayload := base64.StdEncoding.EncodeToString(jsonPayload)

	formatAnnotation := fmt.Sprintf(PayloadAnnotation, payloadType)
	// Use patch instead of update to prevent race conditions.
	patchBytes, err := patch.GetAnnotationsPatch(map[string]string{
		formatAnnotation: textPayload,
	})
	if err != nil {
		return err
	}
	if _, err := b.pipelienclientset.TektonV1beta1().TaskRuns(tr.Namespace).Patch(tr.Name, types.MergePatchType, patchBytes); err != nil {
		return err
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendTekton
}
