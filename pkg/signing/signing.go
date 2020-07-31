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

type TaskRunSigner struct {
	Logger            *zap.SugaredLogger
	Pipelineclientset versioned.Interface
	SecretPath        string
	ConfigStore       config.ConfigGetter
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
	cfg := ts.ConfigStore.Config()
	// First we sign the overall Taskrun as an "artifact"
	rawPayloads := ts.generatePayloads(tr, cfg.Artifacts.TaskRuns.Formats.EnabledFormats)
	ts.Logger.Infof("Generated payloads: %v for %s/%s", rawPayloads, tr.Namespace, tr.Name)

	// Sign all the payloads with all the signing strategies (right now just pgp)
	// TODO: Currently, this reads secrets from disk every time.
	// This could be optimized to instead watch the secret for changes.
	signer, err := pgp.NewSigner(ts.SecretPath, ts.Logger)
	if err != nil {
		return err
	}

	// Signing the payload objects forces them to be marshalled into bytes.
	// Use this marshaled form from now on, instead of the object itself.
	signedPayloads := map[formats.PayloadType][]byte{}
	signatures := map[formats.PayloadType]string{}
	for pt, rp := range rawPayloads {
		ts.Logger.Infof("generating %s payload", pt)
		signature, signed, err := signer.Sign(rp)
		if err != nil {
			ts.Logger.Error(err)
			continue
		}
		signedPayloads[pt] = signed
		signatures[pt] = signature
	}

	// Now store the signature and signed payloads in all the storage backends.
	var merr *multierror.Error
	allBackends := getBackends(ts.Pipelineclientset, ts.Logger, tr)
	enabledBackends := ts.ConfigStore.Config().Artifacts.TaskRuns.StorageBackends.EnabledBackends

	for _, b := range allBackends {
		if _, ok := enabledBackends[b.Type()]; !ok {
			ts.Logger.Debugf("skipping backend %s for taskrun %s/%s", b.Type(), tr.Namespace, tr.Name)
			continue
		}
		for payloadType, signed := range signedPayloads {
			signature := signatures[payloadType]
			// We have the object we signed and the signature for the same payload type. Store both!
			if err := b.StorePayload(signed, signature, payloadType); err != nil {
				ts.Logger.Errorf("error storing payloadType %s on storageBackend %s for taskRun %s/%s: %v", payloadType, b.Type(), tr.Namespace, tr.Name, err)
				merr = multierror.Append(merr, err)
			}
		}
	}
	if merr.ErrorOrNil() != nil {
		return merr
	}

	// TODO: sign any output resources produced (OCI Images, etc.)

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset)
}

func (ts *TaskRunSigner) generatePayloads(tr *v1beta1.TaskRun, enabledFormats map[string]struct{}) map[formats.PayloadType]interface{} {
	payloads := map[formats.PayloadType]interface{}{}
	for _, payloader := range formats.AllPayloadTypes {
		if _, ok := enabledFormats[string(payloader.Type())]; !ok {
			ts.Logger.Debugf("skipping format %s for taskrun %s/%s", payloader, tr.Namespace, tr.Name)
		}
		payload, err := payloader.CreatePayload(tr)
		if err != nil {
			ts.Logger.Errorf("Error creating payload of type %s for %s/%s", payloader, tr.Namespace, tr.Name)
			continue
		}
		payloads[payloader.Type()] = payload
	}
	return payloads
}
