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

package chains

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/signing/kms"
	"github.com/tektoncd/chains/pkg/chains/signing/pgp"
	"github.com/tektoncd/chains/pkg/chains/signing/x509"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/patch"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation = "chains.tekton.dev/signed"
)

type Signer interface {
	SignTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error
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
	if _, err := ps.TektonV1beta1().TaskRuns(tr.Namespace).Patch(
		context.TODO(), tr.Name, types.MergePatchType, patchBytes, v1.PatchOptions{}); err != nil {
		return err
	}
	return nil
}

// Set this as a var for mocking.
var getBackends = storage.InitializeBackends

func allSigners(sp string, cfg config.Config, l *zap.SugaredLogger) map[string]signing.Signer {
	all := map[string]signing.Signer{}
	for _, s := range signing.AllSigners {
		switch s {
		case signing.TypePgp:
			signer, err := pgp.NewSigner(sp, l)
			if err != nil {
				l.Warnf("error configuring pgp signer: %s", err)
				continue
			}
			all[s] = signer
		case signing.TypeX509:
			signer, err := x509.NewSigner(sp, l)
			if err != nil {
				l.Warnf("error configuring x509 signer: %s", err)
				continue
			}
			all[s] = signer
		case signing.TypeKMS:
			signer, err := kms.NewSigner(cfg.Signers.KMS, l)
			if err != nil {
				l.Warnf("error configuring kms signer with config %v: %s", cfg.Signers.KMS, err)
				continue
			}
			all[s] = signer
		default:
			// This should never happen, so panic
			l.Panicf("unsupported signer: %s", s)
		}
	}
	return all
}

// SignTaskRun signs a TaskRun, and marks it as signed.
func (ts *TaskRunSigner) SignTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error {
	// Get all the things we might need (storage backends, signers and formatters)
	cfg := ts.ConfigStore.Config()

	// TODO: Hook this up to config.
	enabledSignableTypes := []artifacts.Signable{
		&artifacts.TaskRunArtifact{Logger: ts.Logger},
		&artifacts.OCIArtifact{Logger: ts.Logger},
	}

	// Storage
	allBackends, err := getBackends(ts.Pipelineclientset, ts.Logger, tr, cfg)
	if err != nil {
		return err
	}

	signers := allSigners(ts.SecretPath, cfg, ts.Logger)

	var merr *multierror.Error
	for _, signableType := range enabledSignableTypes {
		// Extract all the "things" to be signed.
		// We might have a few of each type (several binaries, or images)
		objects := signableType.ExtractObjects(tr)

		// Go through each object one at a time.
		for _, obj := range objects {

			// Find the right payload format and format the object
			payloader, ok := formats.AllPayloadTypes[signableType.PayloadFormat(cfg)]
			if !ok {
				ts.Logger.Warnf("Format %s configured for object: %v %s was not found", signableType.PayloadFormat(cfg), obj, signableType.Type())
				continue
			}
			payload, err := payloader.CreatePayload(obj)
			if err != nil {
				ts.Logger.Error(err)
				continue
			}

			// Sign it!
			signerType := signableType.Signer(cfg)
			signer, ok := signers[signerType]
			if !ok {
				ts.Logger.Warnf("No signer %s configured for object: %v", signerType, obj)
				continue
			}
			if signer == nil {
				ts.Logger.Error("signer is nil")
				continue
			}
			ts.Logger.Infof("Signing object %s with %s", obj, signerType)
			rawPayload, err := json.Marshal(payload)
			if err != nil {
				ts.Logger.Warnf("Unable to marshal payload: %v", signerType, obj)
				continue
			}
			signature, _, err := signer.Sign(ctx, rawPayload)
			if err != nil {
				ts.Logger.Error(err)
				continue
			}

			// Now store those!
			b := allBackends[signableType.StorageBackend(cfg)]
			if err := b.StorePayload(rawPayload, string(signature), signableType.Key(obj)); err != nil {
				ts.Logger.Error(err)
				merr = multierror.Append(merr, err)
			}
		}
		if merr.ErrorOrNil() != nil {
			return merr
		}
	}

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset)
}
