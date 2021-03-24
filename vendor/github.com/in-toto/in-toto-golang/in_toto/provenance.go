package in_toto

import (
	"encoding/json"
	"time"
)

// PostDecodeProvenance computes calculated fields after the JSON
// decoding is done.
func (p *Provenance) PostDecode() error {
	var err error

	// Metadata is OPTIONAL
	if p.Metadata.BuildTimestamp == "" {
		return nil
	}
	p.Metadata.BuildTimestampInternal, err =
		time.Parse(time.RFC3339, p.Metadata.BuildTimestamp)

	return err
}

func (p *Provenance) PreEncode() error {
	// Builds prior to 1970 seems unlikely
	if p.Metadata.BuildTimestampInternal.Unix() < 1 {
		return nil
	}

	p.Metadata.BuildTimestamp =
		p.Metadata.BuildTimestampInternal.Format(time.RFC3339)

	return nil
}

// DecodeProvenanceMem decodes a provenance from JSON and computes
// any dependent fields.
func DecodeProvenanceMem(b []byte) (*Provenance, error) {
	var p Provenance
	var err error

	if err = json.Unmarshal(b, &p); err != nil {
		return nil, err
	}

	if err = p.PostDecode(); err != nil {
		return nil, err
	}

	return &p, nil
}

// DecodeProvenanceString decodes a provenance from JSON and computes
// any dependent fields.
func DecodeProvenanceString(s string) (*Provenance, error) {
	return DecodeProvenanceMem([]byte(s))
}

func EncodeProvenanceString(p *Provenance) (string, error) {
	var err error
	var b []byte

	if err = p.PreEncode(); err != nil {
		return "", err
	}

	if b, err = json.Marshal(p); err != nil {
		return "", err
	}

	return string(b), nil
}
