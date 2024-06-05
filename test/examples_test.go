//go:build e2e
// +build e2e

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

package test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"

	slsa1 "github.com/in-toto/attestation/go/predicates/provenance/v1"
	intoto "github.com/in-toto/attestation/go/v1"
	slsa02 "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	taskRunExamplesPath     = "../examples/taskruns"
	pipelineRunExamplesPath = "../examples/pipelineruns"
)

type TestExample struct {
	name              string
	cm                map[string]string
	pipelinesCm       map[string]string
	getExampleObjects func(t *testing.T, ns string) map[string]objects.TektonObject
	payloadKey        string
	signatureKey      string
	outputLocation    string
	predicate         string
}

// TestExamples copies the format in the tektoncd/pipelines repo
// https://github.com/tektoncd/pipeline/blob/main/test/examples_test.go
func TestExamples(t *testing.T) {
	tests := []TestExample{
		{
			name: "taskrun-examples-slsa-v1",
			cm: map[string]string{
				"artifacts.taskrun.format": "slsa/v1",
				"artifacts.oci.storage":    "tekton",
			},
			getExampleObjects: getTaskRunExamples,
			payloadKey:        "chains.tekton.dev/payload-taskrun-%s",
			signatureKey:      "chains.tekton.dev/signature-taskrun-%s",
			outputLocation:    "slsa/v1",
			predicate:         "slsav0.1",
		},
		{
			name: "pipelinerun-examples-slsa-v1",
			cm: map[string]string{
				"artifacts.pipelinerun.format":  "slsa/v1",
				"artifacts.pipelinerun.storage": "tekton",
			},
			getExampleObjects: getPipelineRunExamples,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
			outputLocation:    "slsa/v1",
			predicate:         "slsav0.1",
		},
		{
			name: "taskrun-examples-slsa-v2alpha3",
			cm: map[string]string{
				"artifacts.taskrun.format": "slsa/v2alpha3",
				"artifacts.oci.storage":    "tekton",
			},
			getExampleObjects: getTaskRunExamples,
			payloadKey:        "chains.tekton.dev/payload-taskrun-%s",
			signatureKey:      "chains.tekton.dev/signature-taskrun-%s",
			outputLocation:    "slsa/v2alpha3",
			predicate:         "slsav1.0",
		},
		{
			name: "pipelinerun-examples-slsa-v2alpha3",
			cm: map[string]string{
				"artifacts.pipelinerun.format": "slsa/v2alpha3",
				"artifacts.oci.storage":        "tekton",
			},
			getExampleObjects: getPipelineRunExamples,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
			outputLocation:    "slsa/v2alpha3",
			predicate:         "slsav1.0",
		},
		{
			name: "taskrun-examples-slsa-v2alpha4",
			cm: map[string]string{
				"artifacts.taskrun.format": "slsa/v2alpha4",
				"artifacts.oci.storage":    "tekton",
			},
			getExampleObjects: getTaskRunExamples,
			payloadKey:        "chains.tekton.dev/payload-taskrun-%s",
			signatureKey:      "chains.tekton.dev/signature-taskrun-%s",
			outputLocation:    "slsa/v2alpha4",
			predicate:         "slsav1.0",
		},
		{
			name: "taskrun-type-hinted-results-v2alpha4",
			cm: map[string]string{
				"artifacts.taskrun.format": "slsa/v2alpha4",
				"artifacts.oci.storage":    "tekton",
			},
			pipelinesCm: map[string]string{
				"enable-api-fields": "alpha",
			},
			getExampleObjects: getTaskRunWithTypeHintedResultsExamples,
			payloadKey:        "chains.tekton.dev/payload-taskrun-%s",
			signatureKey:      "chains.tekton.dev/signature-taskrun-%s",
			outputLocation:    "slsa/v2alpha4",
			predicate:         "slsav1.0",
		},
		{
			name: "pipelinerun-examples-slsa-v2alpha4",
			cm: map[string]string{
				"artifacts.pipelinerun.format": "slsa/v2alpha4",
				"artifacts.oci.storage":        "tekton",
			},
			getExampleObjects: getPipelineRunExamples,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
			outputLocation:    "slsa/v2alpha4",
			predicate:         "slsav1.0",
		},
		{
			name: "pipelinerun-type-hinted-results-v2alpha4",
			cm: map[string]string{
				"artifacts.pipelinerun.format": "slsa/v2alpha4",
				"artifacts.oci.storage":        "tekton",
			},
			pipelinesCm: map[string]string{
				"enable-api-fields": "alpha",
			},
			getExampleObjects: getPipelineRunWithTypeHintedResultsExamples,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
			outputLocation:    "slsa/v2alpha4",
			predicate:         "slsav1.0",
		},
		{
			name: "pipelinerun-no-repeated-subjects-v2alpha4",
			cm: map[string]string{
				"artifacts.pipelinerun.format":                 "slsa/v2alpha4",
				"artifacts.oci.storage":                        "tekton",
				"artifacts.pipelinerun.enable-deep-inspection": "true",
			},
			pipelinesCm: map[string]string{
				"enable-api-fields": "alpha",
			},
			getExampleObjects: getPipelineRunWithRepeatedBuildArtifacts,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
			outputLocation:    "slsa/v2alpha4",
			predicate:         "slsav1.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			cleanUpInTotoFormatter := setConfigMap(ctx, t, c, test.cm)
			t.Cleanup(cleanUpInTotoFormatter)

			if len(test.pipelinesCm) > 0 {
				resetPipelinesConfig := setupPipelinesFeatureFlags(ctx, t, c, test.pipelinesCm)
				t.Cleanup(resetPipelinesConfig)
			}

			runInTotoFormatterTests(ctx, t, ns, c, test)
		})
	}
}

func runInTotoFormatterTests(ctx context.Context, t *testing.T, ns string, c *clients, test TestExample) {
	// TODO: Commenting this out for now. Causes race condition where tests write and revert the chains-config
	// and signing-secrets out of order
	// t.Parallel()
	t.Helper()
	for path, obj := range test.getExampleObjects(t, ns) {
		obj := obj
		t.Run(path, func(t *testing.T) {
			t.Logf("creating object %v", path)

			// create the task run
			createdObj := tekton.CreateObject(t, ctx, c.PipelineClient, obj)

			// give it a minute to complete.
			waitForCondition(ctx, t, c.PipelineClient, createdObj, done, 60*time.Second)

			// now validate the in-toto attestation
			completed := waitForCondition(ctx, t, c.PipelineClient, createdObj, signed, 120*time.Second)
			payload, _ := base64.StdEncoding.DecodeString(completed.GetAnnotations()[fmt.Sprintf(test.payloadKey, completed.GetUID())])
			signature, _ := base64.StdEncoding.DecodeString(completed.GetAnnotations()[fmt.Sprintf(test.signatureKey, completed.GetUID())])
			t.Logf("Got attestation: %s", string(payload))

			if test.predicate == "slsav1.0" {
				// make sure provenance is correct
				var gotProvenance intoto.Statement
				if err := json.Unmarshal(payload, &gotProvenance); err != nil {
					t.Fatal(err)
				}
				expected := expectedProvenanceSLSA1(ctx, t, path, completed, test.outputLocation, ns, c)

				expPredicateStruct := expected.Predicate
				expected.Predicate = nil

				gotPredicateStruct := gotProvenance.Predicate
				gotProvenance.Predicate = nil

				opts := []cmp.Option{
					// Annotations and labels may contain release specific information. Ignore
					// those to avoid brittle tests.
					cmpopts.IgnoreFields(slsa1.BuildDefinition{}, "InternalParameters"),
					cmpopts.IgnoreMapEntries(ignoreEnvironmentAnnotationsAndLabels),
					protocmp.Transform(),
				}

				if diff := cmp.Diff(&expected, &gotProvenance, opts...); diff != "" {
					t.Errorf("provenance dont match: -want +got: %s", diff)
				}

				comparePredicates[slsa1.Provenance](t, expPredicateStruct, gotPredicateStruct, opts)
			} else {
				var gotProvenance intoto.Statement
				if err := json.Unmarshal(payload, &gotProvenance); err != nil {
					t.Fatal(err)
				}
				expected := expectedProvenance(ctx, t, path, completed, test.outputLocation, ns, c)

				expPredicateStruct := expected.Predicate
				expected.Predicate = nil

				gotPredicateStruct := gotProvenance.Predicate
				gotProvenance.Predicate = nil

				opts := []cmp.Option{
					// Annotations and labels may contain release specific information. Ignore
					// those to avoid brittle tests.
					cmpopts.IgnoreFields(slsa02.ProvenanceInvocation{}, "Environment"),
					cmpopts.IgnoreMapEntries(ignoreEnvironmentAnnotationsAndLabels),
					protocmp.Transform(),
				}

				if diff := cmp.Diff(&expected, &gotProvenance, opts...); diff != "" {
					t.Errorf("provenance dont match: -want +got: %s", diff)
				}

				comparePredicates[slsa02.ProvenancePredicate](t, expPredicateStruct, gotPredicateStruct, opts)
			}

			// verify signature
			pub, err := c.secret.x509priv.PublicKey()
			if err != nil {
				t.Fatal(err)
			}
			verifier := &verifier{
				pub: pub.(*ecdsa.PublicKey),
			}
			ev, err := dsse.NewEnvelopeVerifier(verifier)
			if err != nil {
				t.Fatal(err)
			}
			env := dsse.Envelope{}
			if err := json.Unmarshal(signature, &env); err != nil {
				t.Fatal(err)
			}

			if _, err := ev.Verify(ctx, &env); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type verifier struct {
	pub *ecdsa.PublicKey
}

func (v *verifier) Verify(_ context.Context, data, sig []byte) error {
	h := sha256.Sum256(data)
	if ecdsa.VerifyASN1(v.pub, h[:], sig) {
		return nil
	}
	return errors.New("validation error")
}

func (v *verifier) KeyID() (string, error) {
	return "", nil
}

func (v *verifier) Public() crypto.PublicKey {
	return v.pub
}

func expectedProvenanceSLSA1(ctx context.Context, t *testing.T, example string, obj objects.TektonObject, outputLocation string, ns string, c *clients) intoto.Statement {
	t.Helper()
	switch obj.(type) {
	case *objects.TaskRunObjectV1:
		f := expectedTaskRunProvenanceFormat(t, example, obj, outputLocation)
		return expectedAttestationSLSA1(t, example, f, outputLocation)
	case *objects.PipelineRunObjectV1:
		f := expectedPipelineRunProvenanceFormat(ctx, t, obj, ns, c)
		return expectedAttestationSLSA1(t, example, f, outputLocation)
	default:
		t.Error("Unexpected type trying to get provenance")
	}
	return intoto.Statement{}
}

func expectedProvenance(ctx context.Context, t *testing.T, example string, obj objects.TektonObject, outputLocation string, ns string, c *clients) intoto.Statement {
	t.Helper()
	switch obj.(type) {
	case *objects.TaskRunObjectV1:
		f := expectedTaskRunProvenanceFormat(t, example, obj, outputLocation)
		return expectedAttestation(t, example, f, outputLocation)
	case *objects.PipelineRunObjectV1:
		f := expectedPipelineRunProvenanceFormat(ctx, t, obj, ns, c)
		return expectedAttestation(t, example, f, outputLocation)
	default:
		t.Error("Unexpected type trying to get provenance")
	}
	return intoto.Statement{}
}

type URIDigestPair struct {
	URI    string
	Digest string
}

type Format struct {
	Entrypoint         string
	PipelineStartedOn  string
	PipelineFinishedOn string
	UID                string
	BuildStartTimes    []string
	BuildFinishedTimes []string
	ContainerNames     []string
	StepImages         []string
	URIDigest          []URIDigestPair
}

func expectedTaskRunProvenanceFormat(t *testing.T, example string, obj objects.TektonObject, outputLocation string) Format {
	t.Helper()
	tr := obj.GetObject().(*v1.TaskRun)

	name := tr.Name
	if tr.Spec.TaskRef != nil {
		name = tr.Spec.TaskRef.Name
	}

	var stepNames []string
	var images []string
	var uridigest []URIDigestPair
	uriDigestSet := make(map[string]bool)
	for _, step := range tr.Status.Steps {
		stepNames = append(stepNames, step.Name)
		images = append(images, step.ImageID)
		// append uri and digest that havent already been appended
		uri := fmt.Sprintf("oci://%s", strings.Split(step.ImageID, "@")[0])
		digest := strings.Split(step.ImageID, ":")[1]
		uriDigest := fmt.Sprintf("%s %s", uri, digest)
		if _, ok := uriDigestSet[uriDigest]; !ok {
			uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
			uriDigestSet[uriDigest] = true
		}
	}

	return Format{
		Entrypoint:         name,
		UID:                string(tr.ObjectMeta.UID),
		BuildStartTimes:    []string{tr.Status.StartTime.Time.UTC().Format(time.RFC3339)},
		BuildFinishedTimes: []string{tr.Status.CompletionTime.Time.UTC().Format(time.RFC3339)},
		ContainerNames:     stepNames,
		StepImages:         images,
		URIDigest:          uridigest,
	}
}

func expectedPipelineRunProvenanceFormat(ctx context.Context, t *testing.T, obj objects.TektonObject, ns string, c *clients) Format {
	t.Helper()
	pr := obj.GetObject().(*v1.PipelineRun)

	buildStartTimes := []string{}
	buildFinishedTimes := []string{}
	var uridigest []URIDigestPair
	uriDigestSet := make(map[string]bool)

	for _, cr := range pr.Status.ChildReferences {
		taskRun, err := c.PipelineClient.TektonV1().TaskRuns(ns).Get(ctx, cr.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("Did not expect an error but got %v", err)
		}
		buildStartTimes = append(buildStartTimes, taskRun.Status.StartTime.Time.UTC().Format(time.RFC3339))
		buildFinishedTimes = append(buildFinishedTimes, taskRun.Status.CompletionTime.Time.UTC().Format(time.RFC3339))
		for _, step := range taskRun.Status.Steps {
			// append uri and digest that havent already been appended
			uri := fmt.Sprintf("oci://%s", strings.Split(step.ImageID, "@")[0])
			digest := strings.Split(step.ImageID, ":")[1]
			uriDigest := fmt.Sprintf("%s %s", uri, digest)
			if _, ok := uriDigestSet[uriDigest]; !ok {
				uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
				uriDigestSet[uriDigest] = true
			}
		}
		for _, sidecar := range taskRun.Status.Sidecars {
			// append uri and digest that havent already been appended
			uri := fmt.Sprintf("oci://%s", strings.Split(sidecar.ImageID, "@")[0])
			digest := strings.Split(sidecar.ImageID, ":")[1]
			uriDigest := fmt.Sprintf("%s %s", uri, digest)
			if _, ok := uriDigestSet[uriDigest]; !ok {
				uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
				uriDigestSet[uriDigest] = true
			}
		}
	}

	return Format{
		UID:                string(pr.ObjectMeta.UID),
		PipelineStartedOn:  pr.Status.StartTime.Time.UTC().Format(time.RFC3339),
		PipelineFinishedOn: pr.Status.CompletionTime.Time.UTC().Format(time.RFC3339),
		BuildStartTimes:    buildStartTimes,
		BuildFinishedTimes: buildFinishedTimes,
		URIDigest:          uridigest,
	}
}

func expectedAttestationSLSA1(t *testing.T, example string, f Format, outputLocation string) intoto.Statement {
	t.Helper()
	b := readExpectedAttestationBytes(t, example, f, outputLocation)
	return readExpectedAttestationSLSA1(t, b)
}

func expectedAttestation(t *testing.T, example string, f Format, outputLocation string) intoto.Statement {
	t.Helper()
	b := readExpectedAttestationBytes(t, example, f, outputLocation)
	return readExpectedAttestation(t, b)
}

func readExpectedAttestationBytes(t *testing.T, example string, f Format, outputLocation string) *bytes.Buffer {
	t.Helper()
	path := filepath.Join("testdata", outputLocation, strings.Replace(filepath.Base(example), ".yaml", ".json", 1))
	t.Logf("Reading expected provenance from %s", path)
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	tmpl, err := template.New("test").Parse(string(contents))
	if err != nil {
		t.Fatal(err)
	}
	b := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(b, f); err != nil {
		t.Fatal(err)
	}
	t.Logf("Expected attestation: %s", b.String())
	return b
}

func readExpectedAttestationSLSA1(t *testing.T, b *bytes.Buffer) intoto.Statement {
	t.Helper()
	var expected intoto.Statement
	if err := json.Unmarshal(b.Bytes(), &expected); err != nil {
		t.Fatal(err)
	}
	return expected
}

func readExpectedAttestation(t *testing.T, b *bytes.Buffer) intoto.Statement {
	t.Helper()
	var expected intoto.Statement
	if err := json.Unmarshal(b.Bytes(), &expected); err != nil {
		t.Fatal(err)
	}
	return expected
}

func getTaskRunExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	t.Helper()
	examples := make(map[string]objects.TektonObject)
	for _, example := range getExamplePaths(t, taskRunExamplesPath) {
		examples[example] = taskRunFromExample(t, ns, example)
	}
	return examples
}

func getTaskRunWithTypeHintedResultsExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	t.Helper()
	path := "../examples/v2alpha4/task-with-object-type-hinting.yaml"
	trs := make(map[string]objects.TektonObject)
	trs[path] = taskRunFromExample(t, ns, path)
	return trs
}

func getPipelineRunWithTypeHintedResultsExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	t.Helper()
	path := "../examples/v2alpha4/pipeline-with-object-type-hinting.yaml"
	prs := make(map[string]objects.TektonObject)
	prs[path] = pipelineRunFromExample(t, ns, path)
	return prs
}

func getPipelineRunWithRepeatedBuildArtifacts(t *testing.T, ns string) map[string]objects.TektonObject {
	t.Helper()
	path := "../examples/v2alpha4/pipeline-with-repeated-results.yaml"
	prs := make(map[string]objects.TektonObject)
	prs[path] = pipelineRunFromExample(t, ns, path)
	return prs
}

func getPipelineRunExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	t.Helper()
	examples := make(map[string]objects.TektonObject)
	for _, example := range getExamplePaths(t, pipelineRunExamplesPath) {
		examples[example] = pipelineRunFromExample(t, ns, example)
	}
	return examples
}

func getExamplePaths(t *testing.T, dir string) []string {
	t.Helper()
	var examplePaths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.Fatalf("couldn't walk path %s: %v", path, err)
		}
		// Do not append root and any other folders named "examples"
		if info.Name() == "examples" && info.IsDir() {
			return nil
		}
		if info.IsDir() == false && filepath.Ext(info.Name()) == ".yaml" {
			t.Logf("Adding test %s", path)
			examplePaths = append(examplePaths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("couldn't walk example directory %s: %v", dir, err)
	}
	return examplePaths
}

func taskRunFromExample(t *testing.T, ns, example string) objects.TektonObject {
	t.Helper()
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var tr *v1.TaskRun
	if err := yaml.Unmarshal(contents, &tr); err != nil {
		t.Fatal(err)
	}
	tr.Namespace = ns
	return objects.NewTaskRunObjectV1(tr)
}

func pipelineRunFromExample(t *testing.T, ns, example string) objects.TektonObject {
	t.Helper()
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var pr *v1.PipelineRun
	if err := yaml.Unmarshal(contents, &pr); err != nil {
		t.Fatal(err)
	}
	pr.Namespace = ns
	return objects.NewPipelineRunObjectV1(pr)
}

func ignoreEnvironmentAnnotationsAndLabels(key string, value any) bool {
	if key != "environment" {
		return false
	}
	// There are multiple maps with the key "environment", so we must carefully
	// choose the right one.
	switch v := value.(type) {
	case map[string]any:
		_, hasAnnotations := v["annotations"]
		_, hasLabels := v["labels"]
		return hasAnnotations || hasLabels
	}
	return false
}

func comparePredicates[T any](t *testing.T, expPredicateStruct, gotPredicateStruct *structpb.Struct, opts []cmp.Option) {
	t.Helper()
	expJSON, err := expPredicateStruct.MarshalJSON()
	if err != nil {
		t.Fatalf("error getting predicate json: %v", err)
	}

	gotJSON, err := gotPredicateStruct.MarshalJSON()
	if err != nil {
		t.Fatalf("error getting predicate json: %v", err)
	}

	var expectedPredicate T
	json.Unmarshal(expJSON, &expectedPredicate)
	if err != nil {
		t.Fatalf("error getting predicate original struct: %v", err)
	}

	var gotPredicate T
	json.Unmarshal(gotJSON, &gotPredicate)
	if err != nil {
		t.Fatalf("error getting predicate original struct: %v", err)
	}

	if diff := cmp.Diff(&expectedPredicate, &gotPredicate, opts...); diff != "" {
		t.Errorf("predicates dont match: -want +got: %s", diff)
	}
}
