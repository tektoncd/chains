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
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"

	"github.com/ghodss/yaml"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/test/tekton"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	taskRunExamplesPath     = "../examples/taskruns"
	pipelineRunExamplesPath = "../examples/pipelineruns"
)

type TestExample struct {
	name              string
	cm                map[string]string
	getExampleObjects func(t *testing.T, ns string) map[string]objects.TektonObject
	payloadKey        string
	signatureKey      string
}

// TestExamples copies the format in the tektoncd/pipelines repo
// https://github.com/tektoncd/pipeline/blob/main/test/examples_test.go
func TestExamples(t *testing.T) {
	tests := []TestExample{
		{
			name: "taskrun-examples",
			cm: map[string]string{
				"artifacts.taskrun.format": "slsa/v1",
				"artifacts.oci.storage":    "tekton",
			},
			getExampleObjects: getTaskRunExamples,
			payloadKey:        "chains.tekton.dev/payload-taskrun-%s",
			signatureKey:      "chains.tekton.dev/signature-taskrun-%s",
		},
		{
			name: "pipelinerun-examples",
			cm: map[string]string{
				"artifacts.pipelinerun.format":  "slsa/v1",
				"artifacts.pipelinerun.storage": "tekton",
			},
			getExampleObjects: getPipelineRunExamples,
			payloadKey:        "chains.tekton.dev/payload-pipelinerun-%s",
			signatureKey:      "chains.tekton.dev/signature-pipelinerun-%s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			c, ns, cleanup := setup(ctx, t, setupOpts{})
			t.Cleanup(cleanup)

			cleanUpInTotoFormatter := setConfigMap(ctx, t, c, test.cm)
			runInTotoFormatterTests(ctx, t, ns, c, test)
			cleanUpInTotoFormatter()
		})
	}
}

func runInTotoFormatterTests(ctx context.Context, t *testing.T, ns string, c *clients, test TestExample) {

	// TODO: Commenting this out for now. Causes race condition where tests write and revert the chains-config
	// and signing-secrets out of order
	// t.Parallel()

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

			// make sure provenance is correct
			var gotProvenance intoto.ProvenanceStatement
			if err := json.Unmarshal(payload, &gotProvenance); err != nil {
				t.Fatal(err)
			}
			expected := expectedProvenance(t, path, completed)

			opts := []cmp.Option{
				// Annotations and labels may contain release specific information. Ignore
				// those to avoid brittle tests.
				cmpopts.IgnoreFields(slsa.ProvenanceInvocation{}, "Environment"),
				cmpopts.IgnoreMapEntries(ignoreEnvironmentAnnotationsAndLabels),
			}

			if diff := cmp.Diff(expected, gotProvenance, opts...); diff != "" {
				t.Errorf("provenance dont match: -want +got: %s", diff)
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

			if _, err := ev.Verify(&env); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type verifier struct {
	pub *ecdsa.PublicKey
}

func (v *verifier) Verify(data, sig []byte) error {
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

func expectedProvenance(t *testing.T, example string, obj objects.TektonObject) intoto.ProvenanceStatement {
	switch obj.(type) {
	case *objects.TaskRunObject:
		return expectedTaskRunProvenance(t, example, obj)
	case *objects.PipelineRunObject:
		return expectedPipelineRunProvenance(t, example, obj)
	default:
		t.Error("Unexpected type trying to get provenance")
	}
	return intoto.ProvenanceStatement{}
}

type URIDigestPair struct {
	URI    string
	Digest string
}

type Format struct {
	Entrypoint         string
	PipelineStartedOn  string
	PipelineFinishedOn string
	BuildStartTimes    []string
	BuildFinishedTimes []string
	ContainerNames     []string
	StepImages         []string
	URIDigest          []URIDigestPair
}

func expectedTaskRunProvenance(t *testing.T, example string, obj objects.TektonObject) intoto.ProvenanceStatement {
	tr := obj.GetObject().(*v1beta1.TaskRun)

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
		uri := strings.Split(step.ImageID, "@")[0]
		digest := strings.Split(step.ImageID, ":")[1]
		uriDigest := fmt.Sprintf("%s %s", uri, digest)
		if _, ok := uriDigestSet[uriDigest]; !ok {
			uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
			uriDigestSet[uriDigest] = true
		}
	}

	f := Format{
		Entrypoint:         name,
		BuildStartTimes:    []string{tr.Status.StartTime.Time.UTC().Format(time.RFC3339)},
		BuildFinishedTimes: []string{tr.Status.CompletionTime.Time.UTC().Format(time.RFC3339)},
		ContainerNames:     stepNames,
		StepImages:         images,
		URIDigest:          uridigest,
	}

	return readExpectedAttestation(t, example, f)
}

func expectedPipelineRunProvenance(t *testing.T, example string, obj objects.TektonObject) intoto.ProvenanceStatement {
	pr := obj.GetObject().(*v1beta1.PipelineRun)

	buildStartTimes := []string{}
	buildFinishedTimes := []string{}
	var uridigest []URIDigestPair
	uriDigestSet := make(map[string]bool)

	// TODO: Load TaskRun data from ChildReferences.
	for _, trStatus := range pr.Status.TaskRuns {
		buildStartTimes = append(buildStartTimes, trStatus.Status.StartTime.Time.UTC().Format(time.RFC3339))
		buildFinishedTimes = append(buildFinishedTimes, trStatus.Status.CompletionTime.Time.UTC().Format(time.RFC3339))
		for _, step := range trStatus.Status.Steps {
			// append uri and digest that havent already been appended
			uri := strings.Split(step.ImageID, "@")[0]
			digest := strings.Split(step.ImageID, ":")[1]
			uriDigest := fmt.Sprintf("%s %s", uri, digest)
			if _, ok := uriDigestSet[uriDigest]; !ok {
				uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
				uriDigestSet[uriDigest] = true
			}
		}
		for _, sidecar := range trStatus.Status.Sidecars {
			// append uri and digest that havent already been appended
			uri := strings.Split(sidecar.ImageID, "@")[0]
			digest := strings.Split(sidecar.ImageID, ":")[1]
			uriDigest := fmt.Sprintf("%s %s", uri, digest)
			if _, ok := uriDigestSet[uriDigest]; !ok {
				uridigest = append(uridigest, URIDigestPair{URI: uri, Digest: digest})
				uriDigestSet[uriDigest] = true
			}
		}
	}

	f := Format{
		PipelineStartedOn:  pr.Status.StartTime.Time.UTC().Format(time.RFC3339),
		PipelineFinishedOn: pr.Status.CompletionTime.Time.UTC().Format(time.RFC3339),
		BuildStartTimes:    buildStartTimes,
		BuildFinishedTimes: buildFinishedTimes,
		URIDigest:          uridigest,
	}

	return readExpectedAttestation(t, example, f)
}

func readExpectedAttestation(t *testing.T, example string, f Format) intoto.ProvenanceStatement {
	path := filepath.Join("testdata/intoto", strings.Replace(filepath.Base(example), ".yaml", ".json", 1))
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
	var expected intoto.ProvenanceStatement
	if err := json.Unmarshal(b.Bytes(), &expected); err != nil {
		t.Fatal(err)
	}
	return expected
}

func getTaskRunExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	examples := make(map[string]objects.TektonObject)
	for _, example := range getExamplePaths(t, taskRunExamplesPath) {
		examples[example] = taskRunFromExample(t, ns, example)
	}
	return examples
}

func getPipelineRunExamples(t *testing.T, ns string) map[string]objects.TektonObject {
	examples := make(map[string]objects.TektonObject)
	for _, example := range getExamplePaths(t, pipelineRunExamplesPath) {
		examples[example] = pipelineRunFromExample(t, ns, example)
	}
	return examples
}

func getExamplePaths(t *testing.T, dir string) []string {
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
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var tr *v1beta1.TaskRun
	if err := yaml.Unmarshal(contents, &tr); err != nil {
		t.Fatal(err)
	}
	tr.Namespace = ns
	return objects.NewTaskRunObject(tr)
}

func pipelineRunFromExample(t *testing.T, ns, example string) objects.TektonObject {
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var pr *v1beta1.PipelineRun
	if err := yaml.Unmarshal(contents, &pr); err != nil {
		t.Fatal(err)
	}
	pr.Namespace = ns
	return objects.NewPipelineRunObject(pr)
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
