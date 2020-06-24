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
	"github.com/tektoncd/chains/pkg/signing/formats"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation = "chains.tekton.dev/signed"
)

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
	if tr.ObjectMeta.Annotations == nil {
		tr.ObjectMeta.Annotations = map[string]string{}
	}
	tr.ObjectMeta.Annotations[ChainsAnnotation] = "true"
	if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Update(tr); err != nil {
		return err
	}
	return nil
}

// SignTaskRun signs a TaskRun, and marks it as signed.
// Use var for mocking
var SignTaskRun = func(logger *zap.SugaredLogger, ps versioned.Interface, tr *v1beta1.TaskRun) error {
	// First sign the overall TaskRun with all the configured payloads
	payloads := generatePayloads(logger, tr)
	logger.Infof("Generated payloads: %v for %s/%s", payloads, tr.Namespace, tr.Name)

	// TODO: Store those payloads in all the configured storage systems.

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ps)
}

func generatePayloads(logger *zap.SugaredLogger, tr *v1beta1.TaskRun) []interface{} {
	payloads := []interface{}{}
	for _, payloader := range formats.AllPayloadTypes {
		payload, err := payloader.CreatePayload(tr)
		if err != nil {
			logger.Errorf("Error creating payload of type %s for %s/%s", payloader, tr.Namespace, tr.Name)
			continue
		}
		payloads = append(payloads, payload)
	}
	return payloads
}
