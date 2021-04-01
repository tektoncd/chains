package in_toto

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var provenanceData1 = `
{
  "attestation_type": "https://in-toto.io/Provenance/v1",
  "subject": {
    "curl-7.72.0.tar.bz2": { "sha256": "ad91970864102a59765e20ce16216efc9d6ad381471f7accceceab7d905703ef" },
    "curl-7.72.0.tar.gz":  { "sha256": "d4d5899a3868fbb6ae1856c3e55a32ce35913de3956d1973caccd37bd0174fa2" },
    "curl-7.72.0.tar.xz":  { "sha256": "0ded0808c4d85f2ee0db86980ae610cc9d165e9ca9da466196cc73c346513713" },
    "curl-7.72.0.zip":     { "sha256": "e363cc5b4e500bfc727106434a2578b38440aa18e105d57576f3d8f2abebf888" }
  },
  "materials": {
    "git+https://github.com/curl/curl@curl-7_72_0": { "git_commit": "9d954e49bce3706a9a2efb119ecd05767f0f2a9e" },
    "github_hosted_vm:ubuntu-18.04:20210123.1": null,
    "git+https://github.com/actions/checkout@v2":        { "git_commit": "5a4ac9002d0be2fb38bd78e4b4dbde5606d7042f" },
    "git+https://github.com/actions/upload-artifact@v2": { "git_commit": "e448a9b857ee2131e752b06002bf0e093c65e571" },
    "pkg:deb/debian/stunnel4@5.50-3?arch=amd64":               { "sha256": "e1731ae217fcbc64d4c00d707dcead45c828c5f762bcf8cc56d87de511e096fa" },
    "pkg:deb/debian/python-impacket@0.9.15-5?arch=all":        { "sha256": "71fa2e67376c8bc03429e154628ddd7b196ccf9e79dec7319f9c3a312fd76469" },
    "pkg:deb/debian/libzstd-dev@1.3.8+dfsg-3?arch=amd64":      { "sha256": "91442b0ae04afc25ab96426761bbdf04b0e3eb286fdfbddb1e704444cb12a625" },
    "pkg:deb/debian/libbrotli-dev@1.0.7-2+deb10u1?arch=amd64": { "sha256": "05b6e467173c451b6211945de47ac0eda2a3dccb3cc7203e800c633f74de8b4f" }
  },
  "builder": { "id": "https://github.com/Attestations/GitHubHostedActions@v1" },
  "recipe": {
    "type": "https://github.com/Attestations/GitHubActionsWorkflow@v1",
    "material": "git+https://github.com/curl/curl@curl-7_72_0",
    "entry_point": "build.yaml:maketgz"
  },
  "metadata": {
    "build_timestamp": "2020-08-19T08:38:00Z",
    "materials_complete": false
  }
}
`

var provenance1 = Provenance{
	Attestation: Attestation{
		AttestationType: "https://in-toto.io/Provenance/v1",
		Subject: ArtifactCollection{
			"curl-7.72.0.tar.bz2": ArtifactDigest{"sha256": "ad91970864102a59765e20ce16216efc9d6ad381471f7accceceab7d905703ef"},
			"curl-7.72.0.tar.gz":  ArtifactDigest{"sha256": "d4d5899a3868fbb6ae1856c3e55a32ce35913de3956d1973caccd37bd0174fa2"},
			"curl-7.72.0.tar.xz":  ArtifactDigest{"sha256": "0ded0808c4d85f2ee0db86980ae610cc9d165e9ca9da466196cc73c346513713"},
			"curl-7.72.0.zip":     ArtifactDigest{"sha256": "e363cc5b4e500bfc727106434a2578b38440aa18e105d57576f3d8f2abebf888"},
		},
		Materials: ArtifactCollection{
			"git+https://github.com/curl/curl@curl-7_72_0":            ArtifactDigest{"git_commit": "9d954e49bce3706a9a2efb119ecd05767f0f2a9e"},
			"github_hosted_vm:ubuntu-18.04:20210123.1":                nil,
			"git+https://github.com/actions/checkout@v2":              ArtifactDigest{"git_commit": "5a4ac9002d0be2fb38bd78e4b4dbde5606d7042f"},
			"git+https://github.com/actions/upload-artifact@v2":       ArtifactDigest{"git_commit": "e448a9b857ee2131e752b06002bf0e093c65e571"},
			"pkg:deb/debian/stunnel4@5.50-3?arch=amd64":               ArtifactDigest{"sha256": "e1731ae217fcbc64d4c00d707dcead45c828c5f762bcf8cc56d87de511e096fa"},
			"pkg:deb/debian/python-impacket@0.9.15-5?arch=all":        ArtifactDigest{"sha256": "71fa2e67376c8bc03429e154628ddd7b196ccf9e79dec7319f9c3a312fd76469"},
			"pkg:deb/debian/libzstd-dev@1.3.8+dfsg-3?arch=amd64":      ArtifactDigest{"sha256": "91442b0ae04afc25ab96426761bbdf04b0e3eb286fdfbddb1e704444cb12a625"},
			"pkg:deb/debian/libbrotli-dev@1.0.7-2+deb10u1?arch=amd64": ArtifactDigest{"sha256": "05b6e467173c451b6211945de47ac0eda2a3dccb3cc7203e800c633f74de8b4f"},
		},
	},
	Builder: Builder{
		ID: "https://github.com/Attestations/GitHubHostedActions@v1",
	},
	Recipe: Recipe{
		Type:       "https://github.com/Attestations/GitHubActionsWorkflow@v1",
		Material:   "git+https://github.com/curl/curl@curl-7_72_0",
		EntryPoint: "build.yaml:maketgz",
	},
	Metadata: Metadata{
		BuildTimestamp:    &Time{time.Unix(1597826280, 0)},
		MaterialsComplete: false,
	},
}

var provenanceData2 = `
{
  "attestation_type": "https://in-toto.io/Provenance/v1",
  "subject": {
    "curl-7.72.0.tar.bz2": { "sha256": "ad91970864102a59765e20ce16216efc9d6ad381471f7accceceab7d905703ef" },
  },
  "materials": {
    "git+https://github.com/curl/curl@curl-7_72_0": { "git_commit": "9d954e49bce3706a9a2efb119ecd05767f0f2a9e" },
  },
  "builder": { "id": "https://github.com/Attestations/GitHubHostedActions@v1" },
  "recipe": {
    "type": "https://github.com/Attestations/GitHubActionsWorkflow@v1",
    "material": "git+https://github.com/curl/curl@curl-7_72_0",
    "entry_point": "build.yaml:maketgz"
  },
  "metadata": {
    "build_timestamp": "2020-08-19T08:38:00Z",
    "materials_complete": false
  }
}
`

var provenance2 = Provenance{
	Attestation: Attestation{
		AttestationType: "https://in-toto.io/Provenance/v1",
		Subject: ArtifactCollection{
			"curl-7.72.0.tar.bz2": ArtifactDigest{"sha256": "ad91970864102a59765e20ce16216efc9d6ad381471f7accceceab7d905703ef"},
		},
		Materials: ArtifactCollection{
			"git+https://github.com/curl/curl@curl-7_72_0": ArtifactDigest{"git_commit": "9d954e49bce3706a9a2efb119ecd05767f0f2a9e"},
		},
	},
	Builder: Builder{
		ID: "https://github.com/Attestations/GitHubHostedActions@v1",
	},
	Recipe: Recipe{
		Type:       "https://github.com/Attestations/GitHubActionsWorkflow@v1",
		Material:   "git+https://github.com/curl/curl@curl-7_72_0",
		EntryPoint: "build.yaml:maketgz",
	},
	Metadata: Metadata{
		BuildTimestamp:    &Time{time.Unix(1597826280, 0)},
		MaterialsComplete: false,
	},
}

// 1597826280
func init() {
	// Make sure all strings formatted are in tz Zulu
	os.Setenv("TZ", "UTC")
}

func TestParseJSON(t *testing.T) {
	var got Provenance

	err := json.Unmarshal([]byte(provenanceData1), &got)
	assert.Nil(t, err, "Failed to unmarshal JSON")
}

func TestEncodeJSON(t *testing.T) {
	var got Provenance

	enc, err := json.Marshal(&provenance2)
	assert.Nil(t, err, "Failed to marshal JSON")

	err = json.Unmarshal(enc, &got)
	assert.Nil(t, err, "Failed to unmarshal JSON")
}

func TestEncodeNilTime(t *testing.T) {
	p := Provenance{}

	_, err := json.Marshal(&p)
	assert.Nil(t, err, "Failed to marhsal JSON")
}

var jsonNullTime = `
{
  "metadata": {"build_timestamp": null}
}
`

var jsonNoTime = `
{
  "metadata": {"materials_complete": true}
}
`

func TestDecodeNilTime(t *testing.T) {
	var got Provenance
	var want = Provenance{}

	err := json.Unmarshal([]byte(jsonNullTime), &got)
	assert.Nil(t, err, "failed to unmarshal JSON")
	assert.Equal(t, &want, &got, "wrong result")
}

func TestDecodeNoTime(t *testing.T) {
	var got Provenance
	var want = Provenance{
		Metadata: Metadata{
			MaterialsComplete: true,
		},
	}

	err := json.Unmarshal([]byte(jsonNoTime), &got)
	assert.Nil(t, err, "failed to unmarshal JSON")
	assert.Equal(t, &want, &got, "wrong result")
}

var jsonErrTime = `
{
  "metadata": {"build_timestamp": "null"}
}
`

func TestDecodeErrTime(t *testing.T) {
	var p Provenance

	err := json.Unmarshal([]byte(jsonErrTime), &p)
	assert.IsType(t, &time.ParseError{}, err, "wrong error")
}

func TestDecodeNoJson(t *testing.T) {
	var ti Time
	var badJSON = "notjson"

	err := ti.UnmarshalJSON([]byte(badJSON))
	assert.IsType(t, &json.SyntaxError{}, err, "wrong error")
}

func TestDecodeNullJson(t *testing.T) {
	var ti Time
	var nullJSON = "null"

	err := ti.UnmarshalJSON([]byte(nullJSON))
	assert.Nil(t, err, "unexpected error")
	assert.Equal(t, time.Time{}, ti.Time, "Wrong time returned")
}

func TestEncodeZero(t *testing.T) {
	var ti Time

	b, err := ti.MarshalJSON()
	assert.Nil(t, err, "unexpected error")
	assert.Equal(t, "null", string(b), "wrong result")
}
