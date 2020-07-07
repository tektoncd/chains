package in_toto

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

/*
validateHexString is used to validate that a string passed to it contains
only valid hexadecimal characters.
*/
func validateHexString(str string) error {
	formatCheck, _ := regexp.MatchString("^[a-fA-F0-9]+$", str)
	if !formatCheck {
		return fmt.Errorf("'%s' is not a valid hex string", str)
	}
	return nil
}

/*
KeyVal contains the actual values of a key, as opposed to key metadata such as
a key identifier or key type.  For RSA keys, the key value is a pair of public
and private keys in PEM format stored as strings.  For public keys the Private
field may be an empty string.
*/
type KeyVal struct {
	Private string `json:"private"`
	Public  string `json:"public"`
}

/*
Key represents a generic in-toto key that contains key metadata, such as an
identifier, supported hash algorithms to create the identifier, the key type
and the supported signature scheme, and the actual key value.
*/
type Key struct {
	KeyId               string   `json:"keyid"`
	KeyIdHashAlgorithms []string `json:"keyid_hash_algorithms"`
	KeyType             string   `json:"keytype"`
	KeyVal              KeyVal   `json:"keyval"`
	Scheme              string   `json:"scheme"`
}

/*
validatePubKey is a general function to validate if a key is a valid public key.
*/
func validatePubKey(key Key) error {
	if err := validateHexString(key.KeyId); err != nil {
		return fmt.Errorf("keyid: %s", err.Error())
	}
	if key.KeyVal.Private != "" {
		return fmt.Errorf("in key '%s': private key found", key.KeyId)
	}
	if key.KeyVal.Public == "" {
		return fmt.Errorf("in key '%s': public key cannot be empty", key.KeyId)
	}
	return nil
}

/*
validateRSAPubKey checks if a passed key is a valid RSA public key.
*/
func validateRSAPubKey(key Key) error {
	if key.KeyType != "rsa" {
		return fmt.Errorf("invalid KeyType for key '%s': should be 'rsa', got"+
			" '%s'", key.KeyId, key.KeyType)
	}
	if key.Scheme != "rsassa-pss-sha256" {
		return fmt.Errorf("invalid scheme for key '%s': should be "+
			"'rsassa-pss-sha256', got: '%s'", key.KeyId, key.Scheme)
	}
	if err := validatePubKey(key); err != nil {
		return err
	}
	return nil
}

/*
Signature represents a generic in-toto signature that contains the identifier
of the Key, which was used to create the signature and the signature data.  The
used signature scheme is found in the corresponding Key.
*/
type Signature struct {
	KeyId string `json:"keyid"`
	Sig   string `json:"sig"`
}

/*
validateSignature is a function used to check if a passed signature is valid,
by inspecting the key ID and the signature itself.
*/
func validateSignature(signature Signature) error {
	if err := validateHexString(signature.KeyId); err != nil {
		return fmt.Errorf("keyid: %s", err.Error())
	}
	if err := validateHexString(signature.Sig); err != nil {
		return fmt.Errorf("signature with keyid '%s': %s", signature.KeyId,
			err.Error())
	}
	return nil
}

/*
validateSliceOfSignatures is a helper function used to validate multiple
signatures stored in a slice.
*/
func validateSliceOfSignatures(slice []Signature) error {
	for _, signature := range slice {
		if err := validateSignature(signature); err != nil {
			return err
		}
	}
	return nil
}

/*
Link represents the evidence of a supply chain step performed by a functionary.
It should be contained in a generic Metablock object, which provides
functionality for signing and signature verification, and reading from and
writing to disk.
*/
type Link struct {
	Type        string                 `json:"_type"`
	Name        string                 `json:"name"`
	Materials   map[string]interface{} `json:"materials"`
	Products    map[string]interface{} `json:"products"`
	ByProducts  map[string]interface{} `json:"byproducts"`
	Command     []string               `json:"command"`
	Environment map[string]interface{} `json:"environment"`
}

/*
validateArtifacts is a general function used to validate products and materials.
*/
func validateArtifacts(artifacts map[string]interface{}) error {
	for artifactName, artifact := range artifacts {
		artifactValue := reflect.ValueOf(artifact).MapRange()
		for artifactValue.Next() {
			value := artifactValue.Value().Interface().(string)
			hashType := artifactValue.Key().Interface().(string)
			if err := validateHexString(value); err != nil {
				return fmt.Errorf("in artifact '%s', %s hash value: %s",
					artifactName, hashType, err.Error())
			}
		}
	}
	return nil
}

/*
validateLink is a function used to ensure that a passed item of type Link
matches the necessary format.
*/
func validateLink(link Link) error {
	if link.Type != "link" {
		return fmt.Errorf("invalid type for link '%s': should be 'link'",
			link.Name)
	}

	if err := validateArtifacts(link.Materials); err != nil {
		return fmt.Errorf("in materials of link '%s': %s", link.Name,
			err.Error())
	}

	if err := validateArtifacts(link.Products); err != nil {
		return fmt.Errorf("in products of link '%s': %s", link.Name,
			err.Error())
	}

	return nil
}

/*
LinkNameFormat represents a format string used to create the filename for a
signed Link (wrapped in a Metablock). It consists of the name of the link and
the first 8 characters of the signing key id.  LinkNameFormatShort is for links
that are not signed, e.g.:

	fmt.Sprintf(LinkNameFormat, "package",
	"2f89b9272acfc8f4a0a0f094d789fdb0ba798b0fe41f2f5f417c12f0085ff498")
	// returns "package.2f89b9272.link"

	fmt.Sprintf(LinkNameFormatShort, "unsigned")
	// returns "unsigned.link"

*/
const LinkNameFormat = "%s.%.8s.link"
const LinkNameFormatShort = "%s.link"
const SublayoutLinkDirFormat = "%s.%.8s"

/*
SupplyChainItem summarizes common fields of the two available supply chain
item types, Inspection and Step.
*/
type SupplyChainItem struct {
	Name              string     `json:"name"`
	ExpectedMaterials [][]string `json:"expected_materials"`
	ExpectedProducts  [][]string `json:"expected_products"`
}

/*
validateArtifactRule calls UnpackRule to validate that the passed rule conforms
with any of the available rule formats.
*/
func validateArtifactRule(rule []string) error {
	if _, err := UnpackRule(rule); err != nil {
		return err
	}
	return nil
}

/*
validateSliceOfArtifactRules iterates over passed rules to validate them.
*/
func validateSliceOfArtifactRules(rules [][]string) error {
	for _, rule := range rules {
		if err := validateArtifactRule(rule); err != nil {
			return err
		}
	}
	return nil
}

/*
validateSupplyChainItem is used to validate the common elements found in both
steps and inspections. Here, the function primarily ensures that the name of
a supply chain item isn't empty.
*/
func validateSupplyChainItem(item SupplyChainItem) error {
	if item.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := validateSliceOfArtifactRules(item.ExpectedMaterials); err != nil {
		return fmt.Errorf("invalid material rule: %s", err)
	}
	if err := validateSliceOfArtifactRules(item.ExpectedProducts); err != nil {
		return fmt.Errorf("invalid product rule: %s", err)
	}
	return nil
}

/*
Inspection represents an in-toto supply chain inspection, whose command in the
Run field is executed during final product verification, generating unsigned
link metadata.  Materials and products used/produced by the inspection are
constrained by the artifact rules in the inspection's ExpectedMaterials and
ExpectedProducts fields.
*/
type Inspection struct {
	Type string   `json:"_type"`
	Run  []string `json:"run"`
	SupplyChainItem
}

/*
validateInspection ensures that a passed inspection is valid and matches the
necessary format of an inspection.
*/
func validateInspection(inspection Inspection) error {
	if err := validateSupplyChainItem(inspection.SupplyChainItem); err != nil {
		return fmt.Errorf("inspection %s", err.Error())
	}
	if inspection.Type != "inspection" {
		return fmt.Errorf("invalid Type value for inspection '%s': should be "+
			"'inspection'", inspection.SupplyChainItem.Name)
	}
	return nil
}

/*
Step represents an in-toto step of the supply chain performed by a functionary.
During final product verification in-toto looks for corresponding Link
metadata, which is used as signed evidence that the step was performed
according to the supply chain definition.  Materials and products used/produced
by the step are constrained by the artifact rules in the step's
ExpectedMaterials and ExpectedProducts fields.
*/
type Step struct {
	Type            string   `json:"_type"`
	PubKeys         []string `json:"pubkeys"`
	ExpectedCommand []string `json:"expected_command"`
	Threshold       int      `json:"threshold"`
	SupplyChainItem
}

/*
validateStep ensures that a passed step is valid and matches the
necessary format of an step.
*/
func validateStep(step Step) error {
	if err := validateSupplyChainItem(step.SupplyChainItem); err != nil {
		return fmt.Errorf("step %s", err.Error())
	}
	if step.Type != "step" {
		return fmt.Errorf("invalid Type value for step '%s': should be 'step'",
			step.SupplyChainItem.Name)
	}
	for _, keyId := range step.PubKeys {
		if err := validateHexString(keyId); err != nil {
			return fmt.Errorf("in step '%s', keyid: %s",
				step.SupplyChainItem.Name, err.Error())
		}
	}
	return nil
}

/*
ISO8601DateSchema defines the format string of a timestamp following the
ISO 8601 standard.
*/
const ISO8601DateSchema = "2006-01-02T15:04:05Z"

/*
Layout represents the definition of a software supply chain.  It lists the
sequence of steps required in the software supply chain and the functionaries
authorized to perform these steps.  Functionaries are identified by their
public keys.  In addition, the layout may list a sequence of inspections that
are executed during in-toto supply chain verification.  A layout should be
contained in a generic Metablock object, which provides functionality for
signing and signature verification, and reading from and writing to disk.
*/
type Layout struct {
	Type    string         `json:"_type"`
	Steps   []Step         `json:"steps"`
	Inspect []Inspection   `json:"inspect"`
	Keys    map[string]Key `json:"keys"`
	Expires string         `json:"expires"`
	Readme  string         `json:"readme"`
}

// Go does not allow to pass `[]T` (slice with certain type) to a function
// that accepts `[]interface{}` (slice with generic type)
// We have to manually create the interface slice first, see
// https://golang.org/doc/faq#convert_slice_of_interface
// TODO: Is there a better way to do polymorphism for steps and inspections?
func (l *Layout) StepsAsInterfaceSlice() []interface{} {
	stepsI := make([]interface{}, len(l.Steps))
	for i, v := range l.Steps {
		stepsI[i] = v
	}
	return stepsI
}
func (l *Layout) InspectAsInterfaceSlice() []interface{} {
	inspectionsI := make([]interface{}, len(l.Inspect))
	for i, v := range l.Inspect {
		inspectionsI[i] = v
	}
	return inspectionsI
}

/*
validateLayout is a function used to ensure that a passed item of type Layout
matches the necessary format.
*/
func validateLayout(layout Layout) error {
	if layout.Type != "layout" {
		return fmt.Errorf("invalid Type value for layout: should be 'layout'")
	}

	if _, err := time.Parse(ISO8601DateSchema, layout.Expires); err != nil {
		return fmt.Errorf("expiry time parsed incorrectly - date either" +
			" invalid or of incorrect format")
	}

	for keyId, key := range layout.Keys {
		if key.KeyId != keyId {
			return fmt.Errorf("invalid key found")
		}
		if err := validateRSAPubKey(key); err != nil {
			return err
		}
	}

	var namesSeen = make(map[string]bool)
	for _, step := range layout.Steps {
		if namesSeen[step.Name] {
			return fmt.Errorf("non unique step or inspection name found")
		} else {
			namesSeen[step.Name] = true
		}
		if err := validateStep(step); err != nil {
			return err
		}
	}
	for _, inspection := range layout.Inspect {
		if namesSeen[inspection.Name] {
			return fmt.Errorf("non unique step or inspection name found")
		} else {
			namesSeen[inspection.Name] = true
		}
	}
	return nil
}

/*
Metablock is a generic container for signable in-toto objects such as Layout
or Link.  It has two fields, one that contains the signable object and one that
contains corresponding signatures.  Metablock also provides functionality for
signing and signature verification, and reading from and writing to disk.
*/
type Metablock struct {
	// NOTE: Whenever we want to access an attribute of `Signed` we have to
	// perform type assertion, e.g. `metablock.Signed.(Layout).Keys`
	// Maybe there is a better way to store either Layouts or Links in `Signed`?
	// The notary folks seem to have separate container structs:
	// https://github.com/theupdateframework/notary/blob/master/tuf/data/root.go#L10-L14
	// https://github.com/theupdateframework/notary/blob/master/tuf/data/targets.go#L13-L17
	// I implemented it this way, because there will be several functions that
	// receive or return a Metablock, where the type of Signed has to be inferred
	// on runtime, e.g. when iterating over links for a layout, and a link can
	// turn out to be a layout (sublayout)
	Signed     interface{} `json:"signed"`
	Signatures []Signature `json:"signatures"`
}

/*
checkRequiredJsonFields checks that the passed map (obj) has keys for each of
the json tags in the passed struct type (typ), and returns an error otherwise.
*/
func checkRequiredJsonFields(obj map[string]interface{},
	typ reflect.Type) error {

	// Create list of json tags, e.g. `json:"_type"`
	attributeCount := typ.NumField()
	allFields := make([]string, 0)
	for i := 0; i < attributeCount; i++ {
		allFields = append(allFields, typ.Field(i).Tag.Get("json"))
	}

	// Assert that there's a key in the passed map for each tag
	for _, field := range allFields {
		if _, ok := obj[field]; !ok {
			return fmt.Errorf("required field %s missing", field)
		}
	}
	return nil
}

/*
Load parses JSON formatted metadata at the passed path into the Metablock
object on which it was called.  It returns an error if it cannot parse
a valid JSON formatted Metablock that contains a Link or Layout.
*/
func (mb *Metablock) Load(path string) (err error) {
	// Open file and close before returning
	jsonFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := jsonFile.Close(); closeErr != nil {
			err = closeErr
		}
	}()

	// Read entire file
	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	// Unmarshal JSON into a map of raw messages (signed and signatures)
	// We can't fully unmarshal immediately, because we need to inspect the
	// type (link or layout) to decide which data structure to use
	var rawMb map[string]*json.RawMessage
	if err := json.Unmarshal(jsonBytes, &rawMb); err != nil {
		return err
	}

	// Error out on missing `signed` or `signatures` field or if
	// one of them has a `null` value, which would lead to a nil pointer
	// dereference in Unmarshal below.
	if rawMb["signed"] == nil || rawMb["signatures"] == nil {
		return fmt.Errorf("In-toto metadata requires 'signed' and" +
			" 'signatures' parts")
	}

	// Fully unmarshal signatures part
	if err := json.Unmarshal(*rawMb["signatures"], &mb.Signatures); err != nil {
		return err
	}

	// Temporarily copy signed to opaque map to inspect the `_type` of signed
	// and create link or layout accordingly
	var signed map[string]interface{}
	if err := json.Unmarshal(*rawMb["signed"], &signed); err != nil {
		return err
	}

	if signed["_type"] == "link" {
		var link Link
		if err := checkRequiredJsonFields(signed, reflect.TypeOf(link)); err != nil {
			return err
		}

		data, err := rawMb["signed"].MarshalJSON()
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&link); err != nil {
			return err
		}

		mb.Signed = link

	} else if signed["_type"] == "layout" {
		var layout Layout
		if err := checkRequiredJsonFields(signed, reflect.TypeOf(layout)); err != nil {
			return err
		}

		data, err := rawMb["signed"].MarshalJSON()
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&layout); err != nil {
			return err
		}

		mb.Signed = layout

	} else {
		return fmt.Errorf("The '_type' field of the 'signed' part of in-toto" +
			" metadata must be one of 'link' or 'layout'")
	}

	return nil
}

/*
Dump JSON serializes and writes the Metablock on which it was called to the
passed path.  It returns an error if JSON serialization or writing fails.
*/
func (mb *Metablock) Dump(path string) error {
	// JSON encode Metablock formatted with newlines and indentation
	// TODO: parametrize format
	jsonBytes, err := json.MarshalIndent(mb, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON bytes to the passed path with permissions (-rw-r--r--)
	err = ioutil.WriteFile(path, jsonBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

/*
GetSignableRepresentation returns the canonical JSON representation of the
Signed field of the Metablock on which it was called.  If canonicalization
fails the first return value is nil and the second return value is the error.
*/
func (mb *Metablock) GetSignableRepresentation() ([]byte, error) {
	return EncodeCanonical(mb.Signed)
}

/*
VerifySignature verifies the first signature, corresponding to the passed Key,
that it finds in the Signatures field of the Metablock on which it was called.
It returns an error if Signatures does not contain a Signature corresponding to
the passed Key, the object in Signed cannot be canonicalized, or the Signature
is invalid.
*/
func (mb *Metablock) VerifySignature(key Key) error {
	var sig Signature
	for _, s := range mb.Signatures {
		if s.KeyId == key.KeyId {
			sig = s
			break
		}
	}

	if sig == (Signature{}) {
		return fmt.Errorf("No signature found for key '%s'", key.KeyId)
	}

	dataCanonical, err := mb.GetSignableRepresentation()
	if err != nil {
		return err
	}

	if err := VerifySignature(key, sig, dataCanonical); err != nil {
		return err
	}
	return nil
}

/*
validateMetablock ensures that a passed Metablock object is valid. It indirectly
validates the Link or Layout that the Metablock object contains.
*/
func validateMetablock(mb Metablock) error {
	switch mbSignedType := mb.Signed.(type) {
	case Layout:
		if err := validateLayout(mb.Signed.(Layout)); err != nil {
			return err
		}
	case Link:
		if err := validateLink(mb.Signed.(Link)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown type '%s', should be 'layout' or 'link'",
			mbSignedType)
	}

	if err := validateSliceOfSignatures(mb.Signatures); err != nil {
		return fmt.Errorf("validateSignature: %s", err)
	}

	return nil
}

/*
Sign creates a signature over the signed portion of the metablock using the Key
object provided. It then appends the resulting signature to the signatures
field as provided. It returns an error if the Signed object cannot be
canonicalized, or if the key is invalid or not supported.
*/
func (mb *Metablock) Sign(key Key) error {

	dataCanonical, err := mb.GetSignableRepresentation()
	if err != nil {
		return err
	}
	var newSignature Signature

	// FIXME: we could be fancier about signature-generation using a dispatch
	// table or something but for now let's just be explicit
	// (also, lolnogenerics)
	if key.KeyType == "ed25519" && key.Scheme == "ed25519" {
		newSignature, err = GenerateEd25519Signature(dataCanonical, key)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("This key type or signature (%s, %s) scheme is "+
			"not supported yet!", key.KeyType, key.Scheme)
	}

	mb.Signatures = append(mb.Signatures, newSignature)

	return nil
}
