//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcp

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash/crc32"
	"regexp"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/pkg/errors"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type KMS struct {
	client        *kms.KeyManagementClient
	keyResourceID string
	projectID     string
	locationID    string
	keyRing       string
	key           string
	version       string
}

var (
	ErrKMSReference = errors.New("kms specification should be in the format gcpkms://projects/[PROJECT_ID]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[KEY]/versions/[VERSION]")

	re = regexp.MustCompile(`^gcpkms://projects/([^/]+)/locations/([^/]+)/keyRings/([^/]+)/cryptoKeys/([^/]+)(?:/versions/([^/]+))?$`)
)

// schemes for various KMS services are copied from https://github.com/google/go-cloud/tree/master/secrets
const ReferenceScheme = "gcpkms://"

func ValidReference(ref string) error {
	if !re.MatchString(ref) {
		return ErrKMSReference
	}
	return nil
}

func parseReference(resourceID string) (projectID, locationID, keyRing, keyName, version string, err error) {
	v := re.FindStringSubmatch(resourceID)
	if len(v) != 6 {
		err = errors.Errorf("invalid gcpkms format %q", resourceID)
		return
	}
	projectID, locationID, keyRing, keyName, version = v[1], v[2], v[3], v[4], v[5]
	return
}

func NewGCP(ctx context.Context, keyResourceID string) (*KMS, error) {
	projectID, locationID, keyRing, keyName, version, err := parseReference(keyResourceID)
	if err != nil {
		return nil, err
	}

	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new gcp kms client")
	}
	return &KMS{
		client:        client,
		keyResourceID: keyResourceID,
		projectID:     projectID,
		locationID:    locationID,
		keyRing:       keyRing,
		key:           keyName,
		version:       version,
	}, nil
}

func (g *KMS) Sign(ctx context.Context, rawPayload []byte) (signature, signed []byte, err error) {
	// Calculate the digest of the message.
	hash := sha256.New()
	if _, err := hash.Write(rawPayload); err != nil {
		return nil, nil, err
	}
	digest := hash.Sum(nil)
	// Optional but recommended: Compute digest's CRC32C.
	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)
	}
	digestCRC32C := crc32c(digest)

	name, err := g.keyVersionName(ctx)
	if err != nil {
		return nil, nil, err
	}
	req := &kmspb.AsymmetricSignRequest{
		Name: name,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest,
			},
		},
		DigestCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}
	result, err := g.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	// Optional, but recommended: perform integrity verification on result.
	// For more details on ensuring E2E in-transit integrity to and from Cloud KMS visit:
	// https://cloud.google.com/kms/docs/data-integrity-guidelines
	if !result.VerifiedDigestCrc32C {
		return nil, nil, fmt.Errorf("AsymmetricSign: request corrupted in-transit")
	}
	if int64(crc32c(result.Signature)) != result.SignatureCrc32C.Value {
		return nil, nil, fmt.Errorf("AsymmetricSign: response corrupted in-transit")
	}
	return result.GetSignature(), digest, nil
}

func (g *KMS) PublicKey(ctx context.Context) (crypto.PublicKey, error) {
	name, err := g.keyVersionName(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "key version")
	}
	// Build the request.
	pkreq := &kmspb.GetPublicKeyRequest{Name: name}
	// Call the API.
	pk, err := g.client.GetPublicKey(ctx, pkreq)
	if err != nil {
		return nil, errors.Wrap(err, "public key")
	}
	p, _ := pem.Decode([]byte(pk.GetPem()))
	if p == nil {
		return nil, errors.New("pem.Decode failed")
	}
	publicKey, err := x509.ParsePKIXPublicKey(p.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse public key")
	}
	return publicKey, nil
}

func (g *KMS) ECDSAPublicKey(ctx context.Context) (*ecdsa.PublicKey, error) {
	k, err := g.PublicKey(ctx)
	if err != nil {
		return nil, err
	}
	pub, ok := k.(*ecdsa.PublicKey)
	if !ok {
		if err != nil {
			return nil, fmt.Errorf("public key was not ECDSA: %#v", k)
		}
	}
	return pub, nil
}

// keyVersionName returns the first key version found for a key in KMS
// TODO: is there a better way to do this?
func (g *KMS) keyVersionName(ctx context.Context) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", g.projectID, g.locationID, g.keyRing, g.key)

	if g.version != "" {
		parent += fmt.Sprintf("/cryptoKeyVersions/%s", g.version)
		return parent, nil
	}

	req := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: parent,
	}
	iterator := g.client.ListCryptoKeyVersions(ctx, req)

	// pick the first key version that is enabled
	var name string
	for {
		kv, err := iterator.Next()
		if err != nil {
			break
		}
		if kv.State == kmspb.CryptoKeyVersion_ENABLED {
			name = kv.GetName()
			break
		}
	}
	if name == "" {
		return "", errors.New("unable to find an enabled key version in GCP KMS, generate one via `cosign generate-key-pair`")
	}
	return name, nil
}

func (g *KMS) CreateKey(ctx context.Context) (*ecdsa.PublicKey, error) {
	if err := g.createKeyRing(ctx); err != nil {
		return nil, errors.Wrap(err, "creating key ring")
	}
	return g.createKey(ctx)
}

func (g *KMS) createKeyRing(ctx context.Context) error {
	getKeyRingRequest := &kmspb.GetKeyRingRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", g.projectID, g.locationID, g.keyRing),
	}
	if result, err := g.client.GetKeyRing(ctx, getKeyRingRequest); err == nil {
		fmt.Printf("Key ring %s already exists in GCP KMS, moving on to creating key.\n", result.GetName())
		// key ring already exists, no need to create
		return err
	}
	// try to create key ring
	createKeyRingRequest := &kmspb.CreateKeyRingRequest{
		Parent:    fmt.Sprintf("projects/%s/locations/%s", g.projectID, g.locationID),
		KeyRingId: g.keyRing,
	}
	result, err := g.client.CreateKeyRing(ctx, createKeyRingRequest)
	fmt.Printf("Created key ring %s in GCP KMS.\n", result.GetName())
	return err
}

func (g *KMS) createKey(ctx context.Context) (*ecdsa.PublicKey, error) {
	name := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", g.projectID, g.locationID, g.keyRing, g.key)
	getKeyRequest := &kmspb.GetCryptoKeyRequest{
		Name: name,
	}
	if result, err := g.client.GetCryptoKey(ctx, getKeyRequest); err == nil {
		fmt.Printf("Key %s already exists in GCP KMS, skipping creation.\n", result.GetName())
		pub, err := g.ECDSAPublicKey(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving public key")
		}
		return pub, nil
	}
	createKeyRequest := &kmspb.CreateCryptoKeyRequest{
		Parent:      fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", g.projectID, g.locationID, g.keyRing),
		CryptoKeyId: g.key,
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256,
			},
		},
	}
	result, err := g.client.CreateCryptoKey(ctx, createKeyRequest)
	if err != nil {
		return nil, errors.Wrap(err, "creating crypto key")
	}
	fmt.Printf("Created key %s in GCP KMS\n", result.GetName())
	pub, err := g.ECDSAPublicKey(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving public key")
	}
	return pub, nil
}

func (g *KMS) Verify(ctx context.Context, payload, signature []byte) error {
	pub, err := g.PublicKey(ctx)
	if err != nil {
		return errors.Wrap(err, "retrieving public key")
	}
	switch k := pub.(type) {
	case *ecdsa.PublicKey:
		h := sha256.Sum256(payload)
		if !ecdsa.VerifyASN1(k, h[:], signature) {
			return errors.New("unable to verify signature")
		}
	default:
		return fmt.Errorf("unknown public key type: %T", k)
	}

	return nil
}
