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
	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/chains/pkg/signing/formats"
	"github.com/tektoncd/chains/pkg/signing/pgp"
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

type configGetter interface {
	Config() config.Config
}

type TaskRunSigner struct {
	Logger            *zap.SugaredLogger
	Pipelineclientset versioned.Interface
	SecretPath        string
	ConfigStore       configGetter
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

// TODO: Hook this up to config.
var enabledSignableTypes = []artifacts.Signable{&artifacts.TaskRunArtifact{}}

// SignTaskRun signs a TaskRun, and marks it as signed.
func (ts *TaskRunSigner) SignTaskRun(tr *v1beta1.TaskRun) error {

	cfg := ts.ConfigStore.Config()
	allBackends := getBackends(ts.Pipelineclientset, ts.Logger, tr)
	var merr *multierror.Error
	for _, signableType := range enabledSignableTypes {
		// Extract all the "things" to be signed.
		// We might have a few of each type (several binaries, or images)
		objects := signableType.ExtractObjects(tr)

		// Go through each object one at a time.
		for _, obj := range objects {

			var payload interface{}
			configuredPayloadType := signableType.PayloadFormat(cfg)
			for _, payloader := range formats.AllPayloadTypes {
				if payloader.Type() != configuredPayloadType {
					continue
				}
				var err error
				payload, err = payloader.CreatePayload(obj)
				if err != nil {
					ts.Logger.Errorf("Error creating payload of type %s", payloader)
					return err
				}
			}

			signer, err := pgp.NewSigner(ts.SecretPath, ts.Logger)
			if err != nil {
				return err
			}
			signature, signed, err := signer.Sign(payload)
			if err != nil {
				ts.Logger.Error(err)
				continue
			}
			// Now store those!
			for _, b := range allBackends {
				if b.Type() != signableType.StorageBackend(cfg) {
					continue
				}
				if err := b.StorePayload(signed, signature, signableType.Key(obj)); err != nil {
					ts.Logger.Error(err)
					merr = multierror.Append(merr, err)
				}
			}
		}
		if merr.ErrorOrNil() != nil {
			return merr
		}
	}

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset)
}
