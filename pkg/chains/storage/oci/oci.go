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

package oci

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/in-toto/in-toto-golang/in_toto"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/oci/mutate"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/sigstore/cosign/pkg/oci/static"
	"github.com/sigstore/cosign/pkg/types"
	"github.com/tektoncd/chains/pkg/chains/formats/simple"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	StorageBackendOCI = "oci"
)

type Backend struct {
	logger *zap.SugaredLogger
	tr     *v1beta1.TaskRun
	cfg    config.Config
	kc     authn.Keychain
	auth   remote.Option
}

// NewStorageBackend returns a new OCI StorageBackend that stores signatures in an OCI registry
func NewStorageBackend(logger *zap.SugaredLogger, client kubernetes.Interface, tr *v1beta1.TaskRun, cfg config.Config) (*Backend, error) {
	kc, err := k8schain.New(context.TODO(), client,
		k8schain.Options{Namespace: tr.Namespace, ServiceAccountName: tr.Spec.ServiceAccountName})
	if err != nil {
		return nil, err
	}

	return &Backend{
		logger: logger,
		tr:     tr,
		cfg:    cfg,
		kc:     kc,
		auth:   remote.WithAuthFromKeychain(kc),
	}, nil
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
	b.logger.Infof("Storing payload on TaskRun %s/%s", b.tr.Namespace, b.tr.Name)

	if storageOpts.PayloadFormat == "simplesigning" {
		format := simple.SimpleContainerImage{}
		if err := json.Unmarshal(rawPayload, &format); err != nil {
			return errors.Wrap(err, "unmarshal simplesigning")
		}
		return b.uploadSignature(format, rawPayload, signature, storageOpts)
	}

	if storageOpts.PayloadFormat == "in-toto" || storageOpts.PayloadFormat == "tekton-provenance" {
		attestation := in_toto.Statement{}
		if err := json.Unmarshal(rawPayload, &attestation); err != nil {
			return errors.Wrap(err, "unmarshal attestation")
		}

		// This can happen if the Task/TaskRun does not adhere to specific naming conventions
		// like *IMAGE_URL that would serve as hints.
		if len(attestation.Subject) == 0 {
			return errors.New("Did not find anything to attest")
		}

		return b.uploadAttestation(attestation, rawPayload, signature, storageOpts)
	}

	return errors.New("OCI storage backend is only supported for OCI images and in-toto attestations")
}

func (b *Backend) uploadSignature(format simple.SimpleContainerImage, rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
	imageName := format.ImageName()
	b.logger.Infof("Uploading %s signature", imageName)
	var opts []name.Option
	if b.cfg.Storage.OCI.Insecure {
		opts = append(opts, name.Insecure)
	}
	ref, err := name.NewDigest(imageName, opts...)
	if err != nil {
		return errors.Wrap(err, "getting digest")
	}
	se, err := ociremote.SignedEntity(ref)
	if err != nil {
		return errors.Wrap(err, "getting signed image")
	}

	sigOpts := []static.Option{}
	if storageOpts.Cert != "" {
		sigOpts = append(sigOpts, static.WithCertChain([]byte(storageOpts.Cert), []byte(storageOpts.Chain)))
	}
	// Create the new signature for this entity.
	b64sig := base64.StdEncoding.EncodeToString([]byte(signature))
	sig, err := static.NewSignature(rawPayload, b64sig, sigOpts...)
	if err != nil {
		return err
	}
	// Attach the signature to the entity.
	newSE, err := mutate.AttachSignatureToEntity(se, sig)
	if err != nil {
		return err
	}
	repo := ref.Repository
	if b.cfg.Storage.OCI.Repository != "" {
		repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
		if err != nil {
			return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
		}
	}
	// Publish the signatures associated with this entity
	if err := ociremote.WriteSignatures(repo, newSE, ociremote.WithRemoteOptions(b.auth)); err != nil {
		return err
	}
	b.logger.Infof("Successfully uploaded signature for %s", imageName)
	return nil
}

func (b *Backend) uploadAttestation(attestation in_toto.Statement, rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
	// upload an attestation for each subject
	b.logger.Info("Starting to upload attestations to OCI ...")
	for _, subj := range attestation.Subject {
		imageName := fmt.Sprintf("%s@sha256:%s", subj.Name, subj.Digest["sha256"])
		b.logger.Infof("Starting attestation upload to OCI for %s...", imageName)
		var opts []name.Option
		if b.cfg.Storage.OCI.Insecure {
			opts = append(opts, name.Insecure)
		}
		ref, err := name.NewDigest(imageName, opts...)
		if err != nil {
			return errors.Wrapf(err, "getting digest for subj %s", imageName)
		}
		repo := ref.Repository
		if b.cfg.Storage.OCI.Repository != "" {
			repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
			if err != nil {
				return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
			}
		}
		se, err := ociremote.SignedEntity(ref)
		if err != nil {
			return errors.Wrap(err, "getting signed image")
		}
		// Create the new attestation for this entity.
		attOpts := []static.Option{static.WithLayerMediaType(types.DssePayloadType)}
		if storageOpts.Cert != "" {
			attOpts = append(attOpts, static.WithCertChain([]byte(storageOpts.Cert), []byte(storageOpts.Chain)))
		}
		att, err := static.NewAttestation([]byte(signature), attOpts...)
		if err != nil {
			return err
		}
		newImage, err := mutate.AttachAttestationToEntity(se, att)
		if err != nil {
			return err
		}
		// Publish the signatures associated with this entity
		if err := ociremote.WriteAttestations(repo, newImage, ociremote.WithRemoteOptions(b.auth)); err != nil {
			return err
		}
		b.logger.Infof("Successfully uploaded attestation for %s", imageName)
	}
	return nil
}

func (b *Backend) Type() string {
	return StorageBackendOCI
}

func (b *Backend) RetrieveSignature(opts config.StorageOpts) (string, error) {
	return "", fmt.Errorf("not implemented")
}

func (b *Backend) RetrievePayload(opts config.StorageOpts) (string, error) {
	return "", fmt.Errorf("not implemented")
}
