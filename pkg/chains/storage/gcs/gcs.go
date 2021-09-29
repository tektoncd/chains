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

package gcs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"cloud.google.com/go/storage"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

const (
	StorageBackendGCS = "gcs"
	// taskrun-$namespace-$name/$key.signature
	SignatureNameFormat = "taskrun-%s-%s/%s.signature"
	// taskrun-$namespace-$name/$key.payload
	PayloadNameFormat = "taskrun-%s-%s/%s.payload"
)

// Backend is a storage backend that stores signed payloads in the TaskRun metadata as an annotation.
// It is stored as base64 encoded JSON.
type Backend struct {
	logger *zap.SugaredLogger
	tr     *v1beta1.TaskRun
	writer gcsWriter
	reader gcsReader
	cfg    config.Config
}

// NewStorageBackend returns a new Tekton StorageBackend that stores signatures on a TaskRun
func NewStorageBackend(logger *zap.SugaredLogger, tr *v1beta1.TaskRun, cfg config.Config) (*Backend, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	bucket := cfg.Storage.GCS.Bucket
	return &Backend{
		logger: logger,
		tr:     tr,
		writer: &writer{client: client, bucket: bucket},
		reader: &reader{client: client, bucket: bucket},
		cfg:    cfg,
	}, nil
}

// StorePayload implements the Payloader interface.
func (b *Backend) StorePayload(rawPayload []byte, signature string, opts config.StorageOpts) error {
	// We need two object names: the signature and the payload. We want to make these unique to the UID, but easy to find based on the
	// name/namespace as well.
	// $bucket/taskrun-$namespace-$name/$key.signature
	// $bucket/taskrun-$namespace-$name/$key.payload
	root := fmt.Sprintf("taskrun-%s-%s", b.tr.Namespace, b.tr.Name)
	sigName := path.Join(root, fmt.Sprintf("%s.signature", opts.Key))
	b.logger.Infof("Storing payload at %s", sigName)

	sigObj := b.writer.GetWriter(sigName)
	if _, err := sigObj.Write([]byte(signature)); err != nil {
		return err
	}
	if err := sigObj.Close(); err != nil {
		return err
	}

	payloadName := path.Join(root, fmt.Sprintf("%s.payload", opts.Key))
	payloadObj := b.writer.GetWriter(payloadName)
	defer payloadObj.Close()
	if _, err := payloadObj.Write(rawPayload); err != nil {
		return err
	}
	if err := payloadObj.Close(); err != nil {
		return err
	}

	if opts.Cert == "" {
		return nil
	}
	certName := path.Join(root, fmt.Sprintf("%s.cert", opts.Key))
	certObj := b.writer.GetWriter(certName)
	defer certObj.Close()
	if _, err := certObj.Write([]byte(opts.Cert)); err != nil {
		return err
	}
	if err := certObj.Close(); err != nil {
		return err
	}

	chainName := path.Join(root, fmt.Sprintf("%s.chain", opts.Key))
	chainObj := b.writer.GetWriter(chainName)
	defer chainObj.Close()
	if _, err := chainObj.Write([]byte(opts.Chain)); err != nil {
		return err
	}
	if err := chainObj.Close(); err != nil {
		return err
	}

	return nil
}

func (b *Backend) Type() string {
	return StorageBackendGCS
}

type gcsWriter interface {
	GetWriter(object string) io.WriteCloser
}

type writer struct {
	client *storage.Client
	bucket string
}

type gcsReader interface {
	GetReader(object string) (io.ReadCloser, error)
}

type reader struct {
	client *storage.Client
	bucket string
}

func (r *writer) GetWriter(object string) io.WriteCloser {
	ctx := context.Background()
	return r.client.Bucket(r.bucket).Object(object).NewWriter(ctx)
}

func (r *reader) GetReader(object string) (io.ReadCloser, error) {
	ctx := context.Background()
	return r.client.Bucket(r.bucket).Object(object).NewReader(ctx)
}

func (b *Backend) RetrieveSignature(opts config.StorageOpts) (string, error) {
	object := fmt.Sprintf(SignatureNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
	return b.retrieveObject(object)
}

func (b *Backend) RetrievePayload(opts config.StorageOpts) (string, error) {
	object := fmt.Sprintf(PayloadNameFormat, b.tr.Namespace, b.tr.Name, opts.Key)
	return b.retrieveObject(object)
}

func (b *Backend) retrieveObject(object string) (string, error) {
	reader, err := b.reader.GetReader(object)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	payload, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
