/*
Copyright 2024 The Tekton Authors
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

package mldsa

import (
	"crypto"
	"errors"
	"io"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/sigstore/sigstore/pkg/signature"
)

// SignerVerifier implements signature.SignerVerifier and crypto.Signer for MLDSA
type SignerVerifier struct {
	priv *mldsa65.PrivateKey
	pub  *mldsa65.PublicKey
}

// LoadSignerVerifier creates a new SignerVerifier from a private key
func LoadSignerVerifier(priv *mldsa65.PrivateKey) (*SignerVerifier, error) {
	if priv == nil {
		return nil, errors.New("private key cannot be nil")
	}

	// Get the public key from the private key
	pub := priv.Public().(*mldsa65.PublicKey)

	return &SignerVerifier{
		priv: priv,
		pub:  pub,
	}, nil
}

// LoadVerifier creates a new SignerVerifier from a public key
func LoadVerifier(pub *mldsa65.PublicKey) (*SignerVerifier, error) {
	if pub == nil {
		return nil, errors.New("public key cannot be nil")
	}

	return &SignerVerifier{
		pub: pub,
	}, nil
}

// Public implements crypto.Signer interface
func (s *SignerVerifier) Public() crypto.PublicKey {
	return s.pub
}

// Sign signs the given data
func (s *SignerVerifier) Sign(data []byte) ([]byte, error) {
	if s.priv == nil {
		return nil, errors.New("private key not available for signing")
	}

	sig := make([]byte, mldsa65.SignatureSize)
	err := mldsa65.SignTo(s.priv, data, nil, false, sig)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// SignWithOpts implements crypto.Signer interface
func (s *SignerVerifier) SignWithOpts(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	// MLDSA doesn't use pre-hashing, so we use the input directly
	return s.Sign(digest)
}

// SignMessage signs a message from a reader
func (s *SignerVerifier) SignMessage(message io.Reader, opts ...signature.SignOption) ([]byte, error) {
	data, err := io.ReadAll(message)
	if err != nil {
		return nil, err
	}

	return s.Sign(data)
}

// Verify verifies the signature against the data
func (s *SignerVerifier) Verify(data, sig []byte) error {
	if s.pub == nil {
		return errors.New("public key not available for verification")
	}

	if len(sig) != mldsa65.SignatureSize {
		return errors.New("invalid signature size")
	}

	if !mldsa65.Verify(s.pub, data, nil, sig) {
		return errors.New("invalid signature")
	}

	return nil
}

// VerifySignature verifies a signature from readers
func (s *SignerVerifier) VerifySignature(signature, message io.Reader, opts ...signature.VerifyOption) error {
	sig, err := io.ReadAll(signature)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(message)
	if err != nil {
		return err
	}

	return s.Verify(data, sig)
}

// PublicKey returns the public key with optional parameters
func (s *SignerVerifier) PublicKey(opts ...signature.PublicKeyOption) (crypto.PublicKey, error) {
	return s.pub, nil
}

// Type returns the key type for SSH and other uses
func (s *SignerVerifier) Type() string {
	return "mldsa65-sha256"
}

// CreateKey generates a new key pair
func (s *SignerVerifier) CreateKey(rand io.Reader) (crypto.PublicKey, crypto.PrivateKey, error) {
	pub, priv, err := mldsa65.GenerateKey(rand)
	if err != nil {
		return nil, nil, err
	}
	return pub, priv, nil
}
