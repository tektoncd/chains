/*
Package ssl implements the Secure Systems Lab signing-spec (sometimes
abbreviated SSL Siging spec.
https://github.com/secure-systems-lab/signing-spec
*/
package ssl

/*
Verifier verifies a complete message against a signature and key.
If the message was hashed prior to signature generation, the verifier
must perform the same steps.
If the key is not recognized ErrUnknownKey shall be returned.
*/
type Verifier interface {
	Verify(keyID string, data, sig []byte) (bool, error)
}

type EnvelopeVerifier struct {
	providers []Verifier
}

func (ev *EnvelopeVerifier) Verify(e *Envelope) (bool, error) {
	if len(e.Signatures) == 0 {
		return false, ErrNoSignature
	}

	// Decode payload (i.e serialized body)
	body, err := b64Decode(e.Payload)
	if err != nil {
		return false, err
	}
	// Generate PAE(payloadtype, serialized body)
	paeEnc, err := PAE([][]byte{
		[]byte(e.PayloadType),
		body,
	})
	if err != nil {
		return false, err
	}

	// If *any* signature is found to be incorrect, the entire verification
	// step fails even if *some* signatures are correct.
	verified := false
	for _, s := range e.Signatures {
		sig, err := b64Decode(s.Sig)
		if err != nil {
			return false, err
		}

		// Loop over the providers. If a provider recognizes the key, we exit
		// the loop and use the result.
		for _, v := range ev.providers {
			ok, err := v.Verify(s.KeyID, paeEnc, sig)
			if err != nil {
				if err == ErrUnknownKey {
					continue
				}
				return false, err
			}

			if !ok {
				return false, nil
			}

			verified = true
			break
		}
	}

	return verified, nil
}

func NewEnvelopeVerifier(p ...Verifier) EnvelopeVerifier {
	ev := EnvelopeVerifier{
		providers: p,
	}
	return ev
}
