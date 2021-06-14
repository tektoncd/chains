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

	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/rekor/cmd/rekor-cli/app"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/tektoncd/chains/pkg/chains/signing"
	"go.uber.org/zap"
)

type rekor struct {
	c      *client.Rekor
	logger *zap.SugaredLogger
}

type rekorClient interface {
	UploadTlog(ctx context.Context, signer signing.Signer, signature, rawPayload []byte) (*models.LogEntryAnon, error)
}

func (r *rekor) UploadTlog(ctx context.Context, signer signing.Signer, signature, rawPayload []byte) (*models.LogEntryAnon, error) {
	pub, err := signer.PublicKey(ctx)
	if err != nil {
		return nil, err
	}
	pem, err := cosign.KeyToPem(pub)
	if err != nil {
		return nil, err
	}

	return cosign.UploadTLog(r.c, signature, rawPayload, pem)

}

// for testing
var getRekor = func(url string, l *zap.SugaredLogger) (rekorClient, error) {
	rekorClient, err := app.GetRekorClient(url)
	if err != nil {
		return nil, err
	}
	return &rekor{
		c:      rekorClient,
		logger: l,
	}, nil
}
