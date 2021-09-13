/*
Copyright 2021 The Tekton Authors
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

package spire

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

// Verify checks if the TaskRun has an SVID cert
// it then verifies the provided signatures against the cert
func Verify(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) error {
	results := tr.Status.TaskRunResults
	if err := validateResults(results, logger); err != nil {
		return errors.Wrap(err, "validating results")
	}
	logger.Info("Successfully verified all results against SPIRE")
	return nil
}

func validateResults(rs []v1beta1.TaskRunResult, logger *zap.SugaredLogger) error {
	resultMap := map[string]v1beta1.TaskRunResult{}
	for _, r := range rs {
		resultMap[r.Name] = r
	}
	svid, ok := resultMap["SVID"]
	if !ok {
		return errors.New("No SVID found")
	}
	block, _ := pem.Decode([]byte(svid.Value))
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("invalid SVID: %s", err)
	}
	var keysToVerify []string
	for key := range resultMap {
		if strings.HasSuffix(key, ".sig") || key == "SVID" {
			continue
		}
		keysToVerify = append(keysToVerify, key)
	}

	if len(keysToVerify) == 0 {
		return errors.New("no results to validate with SPIRE found")
	}

	for _, key := range keysToVerify {
		logger.Infof("Trying to verify result %s against SPIRE", key)
		if err := verifyKey(cert.PublicKey, key, resultMap); err != nil {
			return err
		}
		logger.Infof("Successfully verified result %s against SPIRE", key)

	}
	return nil
}

func verifyKey(pub interface{}, key string, results map[string]v1beta1.TaskRunResult) error {
	signature, ok := results[key+".sig"]
	if !ok {
		return fmt.Errorf("no signature found for %s", key)
	}
	b, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return fmt.Errorf("invalid signature: %s", err)
	}
	h := sha256.Sum256([]byte(results[key].Value))
	// Check val against sig
	switch t := pub.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(t, h[:], b) {
			return errors.New("invalid signature")
		}
		return nil
	case *rsa.PublicKey:
		return rsa.VerifyPKCS1v15(t, crypto.SHA256, h[:], b)
	case ed25519.PublicKey:
		if !ed25519.Verify(t, []byte(results[key].Value), b) {
			return errors.New("invalid signature")
		}
		return nil
	default:
		return fmt.Errorf("unsupported key type: %s", t)
	}
}
