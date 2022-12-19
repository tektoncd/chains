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

package taskrun

import (
	"fmt"
	"strings"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/internal/material"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	"go.uber.org/zap"
)

const (
	uriSeparator    = "@"
	digestSeparator = ":"
)

// AddStepImagesToMaterials adds step images to predicate.materials
func AddStepImagesToMaterials(steps []v1beta1.StepState, mats *[]slsa.ProvenanceMaterial) error {
	for _, stepState := range steps {
		if err := AddImageIDToMaterials(stepState.ImageID, mats); err != nil {
			return err
		}
	}
	return nil
}

// AddSidecarImagesToMaterials adds sidecar images to predicate.materials
func AddSidecarImagesToMaterials(sidecars []v1beta1.SidecarState, mats *[]slsa.ProvenanceMaterial) error {
	for _, sidecarState := range sidecars {
		if err := AddImageIDToMaterials(sidecarState.ImageID, mats); err != nil {
			return err
		}
	}
	return nil
}

// AddImageIDToMaterials converts an imageId with format <uri>@sha256:<digest> and then adds it to a provenance materials.
func AddImageIDToMaterials(imageID string, mats *[]slsa.ProvenanceMaterial) error {
	m := slsa.ProvenanceMaterial{
		Digest: slsa.DigestSet{},
	}
	uriDigest := strings.Split(imageID, uriSeparator)
	if len(uriDigest) == 2 {
		digest := strings.Split(uriDigest[1], digestSeparator)
		if len(digest) == 2 {
			// no point in partially populating the material
			// do it if both conditions are valid.
			m.URI = uriDigest[0]
			m.Digest[digest[0]] = digest[1]
			*mats = append(*mats, m)
		} else {
			return fmt.Errorf("expected imageID %s to be separable by @ and :", imageID)
		}
	} else {
		return fmt.Errorf("expected imageID %s to be separable by @", imageID)
	}
	return nil
}

// materials constructs `predicate.materials` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func materials(tro *objects.TaskRunObject, logger *zap.SugaredLogger) ([]slsa.ProvenanceMaterial, error) {
	var mats []slsa.ProvenanceMaterial

	// add step images
	if err := AddStepImagesToMaterials(tro.Status.Steps, &mats); err != nil {
		return mats, nil
	}

	// add sidecar images
	if err := AddSidecarImagesToMaterials(tro.Status.Sidecars, &mats); err != nil {
		return mats, nil
	}

	// remove duplicate materials
	// TODO : move this logic to the very end after removing
	// the intermediate return statements below in a followup PR.
	mats, err := material.RemoveDuplicateMaterials(mats)
	if err != nil {
		return mats, err
	}

	gitCommit, gitURL := gitInfo(tro)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		mats = append(mats, slsa.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"sha1": gitCommit},
		})
		return mats, nil
	}

	sms := artifacts.RetrieveMaterialsFromStructuredResults(tro, artifacts.ArtifactsInputsResultName, logger)
	mats = append(mats, sms...)

	if tro.Spec.Resources == nil {
		return mats, nil
	}
	// check for a Git PipelineResource
	for _, input := range tro.Spec.Resources.Inputs {
		if input.ResourceSpec == nil || input.ResourceSpec.Type != v1alpha1.PipelineResourceTypeGit {
			continue
		}

		m := slsa.ProvenanceMaterial{
			Digest: slsa.DigestSet{},
		}

		for _, rr := range tro.Status.ResourcesResult {
			if rr.ResourceName != input.Name {
				continue
			}
			if rr.Key == "url" {
				m.URI = attest.SPDXGit(rr.Value, "")
			} else if rr.Key == "commit" {
				m.Digest["sha1"] = rr.Value
			}
		}

		var url string
		var revision string
		for _, param := range input.ResourceSpec.Params {
			if param.Name == "url" {
				url = param.Value
			}
			if param.Name == "revision" {
				revision = param.Value
			}
		}
		m.URI = attest.SPDXGit(url, revision)
		mats = append(mats, m)
	}

	return mats, nil
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tro *objects.TaskRunObject) (commit string, url string) {
	// Scan for git params to use for materials
	if tro.Status.TaskSpec != nil {
		for _, p := range tro.Status.TaskSpec.Params {
			if p.Default == nil {
				continue
			}
			if p.Name == attest.CommitParam {
				commit = p.Default.StringVal
				continue
			}
			if p.Name == attest.URLParam {
				url = p.Default.StringVal
			}
		}
	}

	for _, p := range tro.Spec.Params {
		if p.Name == attest.CommitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == attest.URLParam {
			url = p.Value.StringVal
		}
	}

	for _, r := range tro.Status.TaskRunResults {
		if r.Name == attest.CommitParam {
			commit = r.Value.StringVal
		}
		if r.Name == attest.URLParam {
			url = r.Value.StringVal
		}
	}

	url = attest.SPDXGit(url, "")
	return
}
