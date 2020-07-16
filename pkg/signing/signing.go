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

package signing

import (
	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/chains/pkg/signing/formats"
	"github.com/tektoncd/chains/pkg/signing/storage"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation = "chains.tekton.dev/signed"
)

type Signer interface {
	SignTaskRun(tr *v1beta1.TaskRun) error
}

type TaskRunSigner struct {
	Logger            *zap.SugaredLogger
	Pipelineclientset versioned.Interface
}

// IsSigned determines whether a TaskRun has already been signed.
func IsSigned(tr *v1beta1.TaskRun) bool {
	signature, ok := tr.ObjectMeta.Annotations[ChainsAnnotation]
	if !ok {
		return false
	}
	return signature == "true"
}

// MarkSigned marks a TaskRun as signed.
func MarkSigned(tr *v1beta1.TaskRun, ps versioned.Interface) error {
	// Use patch instead of update to help prevent race conditions.
	patchBytes, err := patch.GetAnnotationsPatch(map[string]string{
		ChainsAnnotation: "true",
	})
	if err != nil {
		return err
	}
	if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Patch(tr.Name, types.MergePatchType, patchBytes); err != nil {
		return err
	}
	return nil
}

// Set this as a var for mocking.
var getBackends = storage.InitializeBackends

// SignTaskRun signs a TaskRun, and marks it as signed.
func (ts *TaskRunSigner) SignTaskRun(tr *v1beta1.TaskRun) error {
	// First sign the overall TaskRun with all the configured payloads
	payloads := generatePayloads(ts.Logger, tr)
	ts.Logger.Infof("Generated payloads: %v for %s/%s", payloads, tr.Namespace, tr.Name)

	backends := getBackends(ts.Pipelineclientset, ts.Logger)
	for _, b := range backends {
		for payloadType, payload := range payloads {
			if err := b.StorePayload(payload, payloadType, tr); err != nil {
				ts.Logger.Errorf("error storing payloadType %s on storageBackend %s for taskRun %s/%s: %v", payloadType, b.Type(), tr.Namespace, tr.Name, err)
				// continue and store others
			}
		}
	}

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset)
}

func generatePayloads(logger *zap.SugaredLogger, tr *v1beta1.TaskRun) map[string]interface{} {
	payloads := map[string]interface{}{}
	for _, payloader := range formats.AllPayloadTypes {
		payload, err := payloader.CreatePayload(tr)
		if err != nil {
			logger.Errorf("Error creating payload of type %s for %s/%s", payloader, tr.Namespace, tr.Name)
			continue
		}
		payloads[payloader.Type()] = payload
	}
	return payloads
}
