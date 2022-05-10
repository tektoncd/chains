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
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"

	"github.com/ghodss/yaml"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	examplesPath = "../examples/taskruns"
)

// TestExamples copies the format in the tektoncd/pipelines repo
// https://github.com/tektoncd/pipeline/blob/main/test/examples_test.go
func TestExamples(t *testing.T) {
	ctx := context.Background()
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	cleanUpInTotoFormatter := setupInTotoFormatter(ctx, t, c)
	runInTotoFormatterTests(ctx, t, ns, c)
	cleanUpInTotoFormatter()
}

func TestExamplesWithSpire(t *testing.T) {
	ctx := context.Background()
	c, ns, cleanup := setup(ctx, t, setupOpts{})
	defer cleanup()

	cleanUpInTotoFormatter := setupInTotoFormatterWithSpire(ctx, t, c)
	runInTotoFormatterTests(ctx, t, ns, c)
	cleanUpInTotoFormatter()
}

func setupInTotoFormatter(ctx context.Context, t *testing.T, c *clients) func() {
	// Setup the right config.
	return setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.format": "in-toto",
		"artifacts.oci.storage":    "tekton",
	})
}

func setupInTotoFormatterWithSpire(ctx context.Context, t *testing.T, c *clients) func() {
	// Setup the right config.
	return setConfigMap(ctx, t, c, map[string]string{
		"artifacts.taskrun.format": "in-toto",
		"artifacts.oci.storage":    "tekton",
		"spire.enabled":            "true",
		"spire.socketPath":         SpireSocketPath,
	})
}

func runInTotoFormatterTests(ctx context.Context, t *testing.T, ns string, c *clients) {
	t.Parallel()
	examples := getExamplePaths(t, examplesPath)
	for _, example := range examples {
		example := example
		t.Run(example, func(t *testing.T) {
			t.Logf("creating taskrun %v", example)
			tr := taskRunFromExample(t, example)

			// create the task run
			taskRun, err := c.PipelineClient.TektonV1beta1().TaskRuns(ns).Create(ctx, tr, metav1.CreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			// give it a minute to complete.
			waitForCondition(ctx, t, c.PipelineClient, taskRun.Name, ns, done, 60*time.Second)

			// now validate the in-toto attestation
			completed := waitForCondition(ctx, t, c.PipelineClient, taskRun.Name, ns, signed, 120*time.Second)
			payload, _ := base64.StdEncoding.DecodeString(completed.Annotations[fmt.Sprintf("chains.tekton.dev/payload-taskrun-%s", completed.UID)])
			signature, _ := base64.StdEncoding.DecodeString(completed.Annotations[fmt.Sprintf("chains.tekton.dev/signature-taskrun-%s", completed.UID)])
			t.Logf("Got attestation: %s", string(payload))

			// make sure provenance is correct
			var gotProvenance intoto.ProvenanceStatement
			if err := json.Unmarshal(payload, &gotProvenance); err != nil {
				t.Fatal(err)
			}
			expected := expectedProvenance(t, example, completed)

			if d := cmp.Diff(gotProvenance, expected); d != "" {
				t.Fatalf("expected and got do not match:\n%v", d)
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

func expectedProvenance(t *testing.T, example string, tr *v1beta1.TaskRun) intoto.ProvenanceStatement {
	path := filepath.Join("testdata/intoto", strings.Replace(filepath.Base(example), ".yaml", ".json", 1))
	t.Logf("Reading expected provenance from %s", path)

	type Format struct {
		Entrypoint      string
		BuildStartedOn  string
		BuildFinishedOn string
		ContainerNames  []string
		StepImages      []string
	}

	name := tr.Name
	if tr.Spec.TaskRef != nil {
		name = tr.Spec.TaskRef.Name
	}

	var stepNames []string
	var images []string
	for _, step := range tr.Status.Steps {
		stepNames = append(stepNames, step.Name)
		images = append(images, step.ImageID)
	}

	f := Format{
		Entrypoint:      name,
		BuildStartedOn:  tr.Status.StartTime.Time.UTC().Format(time.RFC3339),
		BuildFinishedOn: tr.Status.CompletionTime.Time.UTC().Format(time.RFC3339),
		ContainerNames:  stepNames,
		StepImages:      images,
	}

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

func taskRunFromExample(t *testing.T, example string) *v1beta1.TaskRun {
	contents, err := ioutil.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var tr *v1beta1.TaskRun
	if err := yaml.Unmarshal(contents, &tr); err != nil {
		t.Fatal(err)
	}
	return tr
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
