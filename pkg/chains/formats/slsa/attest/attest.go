/*
Copyright 2022 The Tekton Authors

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

package attest

import (
	"fmt"
	"strings"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CommitParam                  = "CHAINS-GIT_COMMIT"
	URLParam                     = "CHAINS-GIT_URL"
	ChainsReproducibleAnnotation = "chains.tekton.dev/reproducible"
)

type StepAttestation struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func Step(step *v1beta1.Step, stepState *v1beta1.StepState) StepAttestation {
	attestation := StepAttestation{}

	entrypoint := strings.Join(step.Command, " ")
	if step.Script != "" {
		entrypoint = step.Script
	}
	attestation.EntryPoint = entrypoint
	attestation.Arguments = step.Args

	env := map[string]interface{}{}
	env["image"] = artifacts.OCIScheme + strings.TrimPrefix(stepState.ImageID, "docker-pullable://")
	env["container"] = stepState.Name
	attestation.Environment = env

	return attestation
}

func Invocation(source *v1beta1.RefSource, params []v1beta1.Param, paramSpecs []v1beta1.ParamSpec, meta metav1.Object) slsa.ProvenanceInvocation {
	i := slsa.ProvenanceInvocation{
		ConfigSource: convertConfigSource(source),
	}
	iParams := make(map[string]v1beta1.ParamValue)

	// get implicit parameters from defaults
	for _, p := range paramSpecs {
		if p.Default != nil {
			iParams[p.Name] = *p.Default
		}
	}

	// get explicit parameters
	for _, p := range params {
		iParams[p.Name] = p.Value
	}

	i.Parameters = iParams

	environment := map[string]map[string]string{}

	annotations := map[string]string{}
	for name, value := range meta.GetAnnotations() {
		// Ignore annotations that are not relevant to provenance information
		if name == corev1.LastAppliedConfigAnnotation || strings.HasPrefix(name, "chains.tekton.dev/") {
			continue
		}
		annotations[name] = value
	}
	if len(annotations) > 0 {
		environment["annotations"] = annotations
	}

	labels := meta.GetLabels()
	if len(labels) > 0 {
		environment["labels"] = labels
	}

	if len(environment) > 0 {
		i.Environment = environment
	}

	return i
}

func convertConfigSource(source *v1beta1.RefSource) slsa.ConfigSource {
	if source == nil {
		return slsa.ConfigSource{}
	}
	return slsa.ConfigSource{
		URI:        source.URI,
		Digest:     source.Digest,
		EntryPoint: source.EntryPoint,
	}
}

// supports the SPDX format which is recommended by in-toto
// ref: https://spdx.dev/spdx-specification-21-web-version/#h.49x2ik5
// ref: https://github.com/in-toto/attestation/blob/849867bee97e33678f61cc6bd5da293097f84c25/spec/field_types.md
func SPDXGit(url, revision string) string {
	if revision == "" {
		return artifacts.GitSchemePrefix + url + ".git"
	}
	return artifacts.GitSchemePrefix + url + fmt.Sprintf("@%s", revision)
}
