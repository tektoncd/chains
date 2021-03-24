package in_toto

import (
	"time"
)

type ArtifactDigest map[string]string
type ArtifactCollection map[string]ArtifactDigest

const (
	ProvenanceTypeV1 = "https://in-toto.io/Provenance/v1"
)

type Attestation struct {
	AttestationType string             `json:"attestation_type"`
	Subject         ArtifactCollection `json:"subject"`
	Materials       ArtifactCollection `json:"materials"`
}

type Builder struct {
	ID string `json:"id"`
}

type Recipe struct {
	Type       string      `json:"type"`
	Material   string      `json:"material,omitempty"`
	EntryPoint string      `json:"entry_point,omitempty"`
	Arguments  interface{} `json:"arguments,omitempty"`
}

type Metadata struct {
	BuildTimestamp         string    `json:"build_timestamp,omitempty"`
	BuildTimestampInternal time.Time `json:"-"`
	MaterialsComplete      bool      `json:"materials_complete,omitempty"`
}

type Provenance struct {
	Attestation
	Builder         Builder     `json:"builder"`
	Recipe          Recipe      `json:"recipe"`
	Reproducibility interface{} `json:"reproducibility,omitempty"`
	Metadata        Metadata    `json:"metadata,omitempty"`
}
