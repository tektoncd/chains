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
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6"
	"github.com/tektoncd/chains/pkg/chains/formats/provenance"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/chains/formats/tekton"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"github.com/tektoncd/chains/pkg/chains/signing/kms"
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
	ChainsAnnotation             = "chains.tekton.dev/signed"
	ChainsTransparencyAnnotation = "chains.tekton.dev/transparency"
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
func MarkSigned(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string) error {
	// Use patch instead of update to help prevent race conditions.
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[ChainsAnnotation] = "true"
	patchBytes, err := patch.GetAnnotationsPatch(annotations)
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

func allFormatters(cfg config.Config, l *zap.SugaredLogger) map[formats.PayloadType]formats.Payloader {
	all := map[formats.PayloadType]formats.Payloader{}

	for _, f := range formats.AllFormatters {
		switch f {
		case formats.PayloadTypeTekton:
			formatter, err := tekton.NewFormatter()
			if err != nil {
				l.Warnf("error configuring tekton formatter: %s", err)
			}
			all[f] = formatter
		case formats.PayloadTypeSimpleSigning:
			formatter, err := simple.NewFormatter()
			if err != nil {
				l.Warnf("error configuring simplesigning formatter: %s", err)
			}
			all[f] = formatter
		case formats.PayloadTypeInTotoIte6:
			formatter, err := intotoite6.NewFormatter(cfg)
			if err != nil {
				l.Warnf("error configuring intoto formatter: %s", err)
			}
			all[f] = formatter
		case formats.PayloadTypeProvenance:
			formatter, err := provenance.NewFormatter(cfg, l)
			if err != nil {
				l.Warnf("error configuring intoto formatter: %s", err)
			}
			all[f] = formatter
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
	allFormats := allFormatters(cfg, ts.Logger)

	rekorClient, err := getRekor(cfg.Transparency.URL, ts.Logger)
	if err != nil {
		return err
	}

	var merr *multierror.Error
	extraAnnotations := map[string]string{}
	for _, signableType := range enabledSignableTypes {
		// Extract all the "things" to be signed.
		// We might have a few of each type (several binaries, or images)
		objects := signableType.ExtractObjects(tr)

		// Go through each object one at a time.
		for _, obj := range objects {
			payloadFormat := signableType.PayloadFormat(cfg)

			// Find the right payload format and format the object
			payloader, ok := allFormats[payloadFormat]
			if !ok {
				ts.Logger.Warnf("Format %s configured for object: %v %s was not found", payloadFormat, obj, signableType.Type())
				continue
			}
			payload, err := payloader.CreatePayload(obj)
			if err != nil {
				ts.Logger.Error(err)
				continue
			}
			ts.Logger.Infof("Created payload of type %s for TaskRun %s/%s", string(payloadFormat), tr.Namespace, tr.Name)

			// Sign it!
			signerType := signableType.Signer(cfg)
			signer, ok := signers[signerType]
			if !ok {
				ts.Logger.Warnf("No signer %s configured for %s: %v", signerType, signableType.Type())
				continue
			}

			if payloader.Wrap() {
				wrapped, err := signing.Wrap(ctx, signer)
				if err != nil {
					return err
				}
				ts.Logger.Infof("Using wrapped envelope signer for %s", payloader.Type())
				signer = wrapped
			}

			ts.Logger.Infof("Signing object with %s", signerType)
			rawPayload, err := json.Marshal(payload)
			if err != nil {
				ts.Logger.Warnf("Unable to marshal payload: %v", signerType, obj)
				continue
			}

			signature, signed, err := signer.Sign(ctx, rawPayload)
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

			if cfg.Transparency.Enabled {
				if payloadFormat == "in-toto" || payloadFormat == "tekton-provenance" {
					entry, err := rekorClient.UploadTlog(ctx, signer, signature, signed)
					if err != nil {
						ts.Logger.Error(err)
						merr = multierror.Append(merr, err)
					} else {
						ts.Logger.Infof("Uploaded entry to %s with index %d", cfg.Transparency.URL, *entry.LogIndex)
						extraAnnotations[ChainsTransparencyAnnotation] = fmt.Sprintf("%s/%d", cfg.Transparency.URL, *entry.LogIndex)
					}
				}
			}
		}
		if merr.ErrorOrNil() != nil {
			return merr
		}
	}

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset, extraAnnotations)
}
