package in_toto

import (
	"encoding/json"
	"time"
)

type ArtifactDigest map[string]string
type ArtifactCollection map[string]ArtifactDigest

const (
	ProvenanceTypeV1 = "https://in-toto.io/Provenance/v1"
)

type Time struct {
	time.Time `json:",inline"`
}

type Attestation struct {
	AttestationType string             `json:"attestation_type,omitempty"`
	Subject         ArtifactCollection `json:"subject,omitempty"`
	Materials       ArtifactCollection `json:"materials,omitempty"`
}

type Builder struct {
	ID string `json:"id,omitempty"`
}

type Recipe struct {
	Type       string      `json:"type,omitempty"`
	Material   string      `json:"material,omitempty"`
	EntryPoint string      `json:"entry_point,omitempty"`
	Arguments  interface{} `json:"arguments,omitempty"`
}

type Metadata struct {
	BuildTimestamp    *Time `json:"build_timestamp,omitempty"`
	MaterialsComplete bool  `json:"materials_complete,omitempty"`
}

type Provenance struct {
	Attestation     `json:",inline"`
	Builder         Builder     `json:"builder,omitempty"`
	Recipe          Recipe      `json:"recipe,omitempty"`
	Reproducibility interface{} `json:"reproducibility,omitempty"`
	Metadata        Metadata    `json:"metadata,omitempty"`
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var err error
	var str string

	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	// Get the real string value
	if err = json.Unmarshal(b, &str); err != nil {
		return err
	}

	t.Time, err = time.Parse(time.RFC3339, str)

	return err
}

func (t *Time) MarshalJSON() ([]byte, error) {
	// Return "null"
	if t.IsZero() {
		return []byte("null"), nil
	}
	// "time"
	buf := make([]byte, 0, len(time.RFC3339)+2)
	buf = append(buf, '"')

	buf = t.UTC().AppendFormat(buf, time.RFC3339)
	buf = append(buf, '"')

	return buf, nil
}
