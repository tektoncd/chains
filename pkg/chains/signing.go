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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

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
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

const (
	// ChainsAnnotation is the standard annotation to indicate a TR has been signed.
	ChainsAnnotation             = "chains.tekton.dev/signed"
	ChainsRetriesAnnotation      = "chains.tekton.dev/retries"
	ChainsTransparencyAnnotation = "chains.tekton.dev/transparency"
	Failed                       = "failed"
	MaxRetries                   = 3
)

type Signer interface {
	SignTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error
}

type TaskRunSigner struct {
	KubeClient        kubernetes.Interface
	Pipelineclientset versioned.Interface
	SecretPath        string
}

// IsSigned determines whether a TaskRun has already been signed.
func IsSigned(tr *v1beta1.TaskRun) (bool, string) {
	const signed = "true"
	signature, ok := tr.ObjectMeta.Annotations[ChainsAnnotation]
	if !ok {
		return false, ""
	}
	return signature == signed, signature
}

// returns true if:
// the signed annotation isn't there, the failed annotation isn't there, and retries<3
func RetryAgain(tr *v1beta1.TaskRun) (bool, string) {
	retries, ok := tr.ObjectMeta.Annotations[ChainsRetriesAnnotation]
	if !ok {
		return false, ""
	}
	return true, retries
}

// MarkSigned marks a TaskRun as signed.
func MarkSigned(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string,
	state string) error {
	return markannotation(tr, ps, annotations, ChainsAnnotation, state)
}

// AddRetries marks a TaskRun as retrying.
func AddRetries(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string,
	value int) error {
	if value > MaxRetries {
		return fmt.Errorf("The retries cannot be greater than %d", MaxRetries)
	}
	return markannotation(tr, ps, annotations, ChainsRetriesAnnotation, strconv.Itoa(value))
}

func markannotation(tr *v1beta1.TaskRun, ps versioned.Interface, annotations map[string]string,
	key, value string) error {
	// Use patch instead of update to help prevent race conditions.
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
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
			signer, err := x509.NewSigner(sp, cfg, l)
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
	cfg := *config.FromContext(ctx)
	logger := logging.FromContext(ctx)

	// TODO: Hook this up to config.
	enabledSignableTypes := []artifacts.Signable{
		&artifacts.TaskRunArtifact{Logger: logger},
		&artifacts.OCIArtifact{Logger: logger},
	}

	// Storage
	allBackends, err := getBackends(ts.Pipelineclientset, ts.KubeClient, logger, tr, cfg)
	if err != nil {
		return err
	}

	signers := allSigners(ts.SecretPath, cfg, logger)
	allFormats := allFormatters(cfg, logger)

	rekorClient, err := getRekor(cfg.Transparency.URL, logger)
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
				logger.Warnf("Format %s configured for object: %v %s was not found", payloadFormat, obj, signableType.Type())
				continue
			}
			payload, err := payloader.CreatePayload(obj)
			if err != nil {
				logger.Error(err)
				continue
			}
			logger.Infof("Created payload of type %s for TaskRun %s/%s", string(payloadFormat), tr.Namespace, tr.Name)

			// Sign it!
			signerType := signableType.Signer(cfg)
			signer, ok := signers[signerType]
			if !ok {
				logger.Warnf("No signer %s configured for %s", signerType, signableType.Type())
				continue
			}

			if payloader.Wrap() {
				wrapped, err := signing.Wrap(ctx, signer)
				if err != nil {
					return err
				}
				logger.Infof("Using wrapped envelope signer for %s", payloader.Type())
				signer = wrapped
			}

			logger.Infof("Signing object with %s", signerType)
			rawPayload, err := json.Marshal(payload)
			if err != nil {
				logger.Warnf("Unable to marshal payload: %v", signerType, obj)
				continue
			}

			signature, err := signer.SignMessage(bytes.NewReader(rawPayload))
			if err != nil {
				logger.Error(err)
				continue
			}

			// Now store those!
			b := allBackends[signableType.StorageBackend(cfg)]
			storageOpts := config.StorageOpts{
				Key:   signableType.Key(obj),
				Cert:  signer.Cert(),
				Chain: signer.Chain(),
			}
			if err := b.StorePayload(rawPayload, string(signature), storageOpts); err != nil {
				logger.Error(err)
				merr = multierror.Append(merr, err)
			}

			if shouldUploadTlog(cfg, tr) {
				entry, err := rekorClient.UploadTlog(ctx, signer, signature, rawPayload, signer.Cert(), string(payloadFormat))
				if err != nil {
					logger.Error(err)
					merr = multierror.Append(merr, err)
				} else {
					logger.Infof("Uploaded entry to %s with index %d", cfg.Transparency.URL, *entry.LogIndex)
					extraAnnotations[ChainsTransparencyAnnotation] = fmt.Sprintf("%s/%d", cfg.Transparency.URL, *entry.LogIndex)
				}
			}
		}
		if merr.ErrorOrNil() != nil {
			if err := HandleRetries(tr, ts, extraAnnotations); err != nil {
				merr = multierror.Append(merr, err)
			}
			return merr
		}
	}

	// Now mark the TaskRun as signed
	return MarkSigned(tr, ts.Pipelineclientset, extraAnnotations, "true")
}

// HandleRetries adds retry annotation to keep track of retries and marks the job as failed
// if retries have reached MaxRetries.
func HandleRetries(tr *v1beta1.TaskRun, ts *TaskRunSigner, extraAnnotations map[string]string) error {
	ok, s := RetryAgain(tr)
	if !ok {
		return AddRetries(tr, ts.Pipelineclientset, extraAnnotations, 1)
	}

	count, err := strconv.Atoi(s)
	if err != nil {
		return err
	}

	if count <= MaxRetries {
		return AddRetries(tr, ts.Pipelineclientset, extraAnnotations, count+1)
	}

	// Marking it failed after retrying MaxRetries
	return MarkSigned(tr, ts.Pipelineclientset, extraAnnotations, Failed)
}
