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
	"errors"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/sigstore/cosign/pkg/cosign"
	rc "github.com/sigstore/rekor/pkg/client"
	"github.com/sigstore/rekor/pkg/generated/client"
	"github.com/sigstore/rekor/pkg/generated/client/entries"
	"github.com/sigstore/rekor/pkg/generated/models"
	intoto_v001 "github.com/sigstore/rekor/pkg/types/intoto/v0.0.1"
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
	pub, err := signer.PublicKey()
	if err != nil {
		return nil, err
	}
	pem, err := cosign.KeyToPem(pub)
	if err != nil {
		return nil, err
	}

	pubKey := strfmt.Base64(pem)
	e := intoto_v001.V001Entry{
		IntotoObj: models.IntotoV001Schema{
			Content: &models.IntotoV001SchemaContent{
				Envelope: string(signature),
			},
			PublicKey: &pubKey,
		},
	}

	entry := models.Intoto{
		APIVersion: swag.String(e.APIVersion()),
		Spec:       e.IntotoObj,
	}
	params := entries.NewCreateLogEntryParams()
	params.SetProposedEntry(&entry)
	resp, err := r.c.Entries.CreateLogEntry(params)
	if err != nil {
		// If the entry already exists, we get a specific error.
		// Here, we display the proof and succeed.
		if existsErr, ok := err.(*entries.CreateLogEntryConflict); ok {

			r.logger.Info("Signature already exists")
			uriSplit := strings.Split(existsErr.Location.String(), "/")
			uuid := uriSplit[len(uriSplit)-1]
			return cosign.VerifyTLogEntry(r.c, uuid)
		}
		return nil, err
	}
	// UUID is at the end of location
	for _, p := range resp.Payload {
		return &p, nil
	}
	return nil, errors.New("bad response from server")
}

// for testing
var getRekor = func(url string, l *zap.SugaredLogger) (rekorClient, error) {
	rekorClient, err := rc.GetRekorClient(url)
	if err != nil {
		return nil, err
	}
	return &rekor{
		c:      rekorClient,
		logger: l,
	}, nil
}
