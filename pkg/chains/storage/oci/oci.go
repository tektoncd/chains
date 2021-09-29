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
	"encoding/json"
	"fmt"

	"github.com/in-toto/in-toto-golang/in_toto"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/cosign"
	cremote "github.com/sigstore/cosign/pkg/cosign/remote"
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
		format := simple.NewSimpleStruct()
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
		return b.uploadAttestation(attestation, rawPayload, signature, storageOpts)
	}

	return errors.New("OCI storage backend is only supported for OCI images and in-toto attestations")
}

func (b *Backend) uploadSignature(format simple.Simple, rawPayload []byte, signature string, storageOpts config.StorageOpts) error {
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
	dgst, err := v1.NewHash(ref.DigestStr())
	if err != nil {
		return errors.Wrap(err, "parsing digest")
	}
	repo := ref.Repository
	if b.cfg.Storage.OCI.Repository != "" {
		repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
		if err != nil {
			return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
		}
	}
	cosignDst := cosign.AttachedImageTag(repo, dgst, cosign.SignatureTagSuffix)
	if err != nil {
		return errors.Wrap(err, "destination ref")
	}
	if _, err = cremote.UploadSignature([]byte(signature), rawPayload, cosignDst, cremote.UploadOpts{
		RemoteOpts: []remote.Option{b.auth},
		Cert:       []byte(storageOpts.Cert),
		Chain:      []byte(storageOpts.Chain),
	}); err != nil {
		return errors.Wrap(err, "uploading")
	}
	b.logger.Infof("Successfully uploaded signature for %s to %s", imageName, cosignDst)
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
		dgst, err := v1.NewHash(ref.DigestStr())
		if err != nil {
			return errors.Wrapf(err, "parsing digest for %s", imageName)
		}
		repo := ref.Repository
		if b.cfg.Storage.OCI.Repository != "" {
			repo, err = name.NewRepository(b.cfg.Storage.OCI.Repository)
			if err != nil {
				return errors.Wrapf(err, "%s is not a valid repository", b.cfg.Storage.OCI.Repository)
			}
		}
		attRef := cosign.AttachedImageTag(repo, dgst, cosign.AttestationTagSuffix)
		if err != nil {
			return errors.Wrapf(err, "destination ref for %s", imageName)
		}
		if _, err = cremote.UploadSignature([]byte{}, []byte(signature), attRef, cremote.UploadOpts{
			RemoteOpts: []remote.Option{b.auth},
			Cert:       []byte(storageOpts.Cert),
			Chain:      []byte(storageOpts.Chain),
			MediaType:  types.DssePayloadType,
		}); err != nil {
			return errors.Wrap(err, "uploading")
		}
		b.logger.Infof("Successfully uploaded attestation for %s to %s", imageName, attRef.String())
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
