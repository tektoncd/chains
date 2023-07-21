/*
Copyright 2023 The Tekton Authors

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

package material

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/internal/backport"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	uriSeparator    = "@"
	digestSeparator = ":"
)

// TaskMaterials constructs `predicate.materials` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func TaskMaterials(ctx context.Context, tro *objects.TaskRunObject) ([]common.ProvenanceMaterial, error) {
	var mats []common.ProvenanceMaterial

	// add step images
	stepMaterials, err := FromStepImages(tro.Status.Steps)
	if err != nil {
		return nil, err
	}
	mats = append(mats, stepMaterials...)

	// add sidecar images
	sidecarMaterials, err := FromSidecarImages(tro.Status.Sidecars)
	if err != nil {
		return nil, err
	}
	mats = append(mats, sidecarMaterials...)

	mats = append(mats, FromTaskParamsAndResults(ctx, tro)...)

	// add task resources
	mats = append(mats, FromTaskResources(ctx, tro)...)

	// remove duplicate materials
	mats, err = removeDuplicateMaterials(mats)
	if err != nil {
		return mats, err
	}
	return mats, nil
}

func PipelineMaterials(ctx context.Context, pro *objects.PipelineRunObject) ([]common.ProvenanceMaterial, error) {
	var mats []common.ProvenanceMaterial
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil {
		m := common.ProvenanceMaterial{
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		mats = append(mats, m)
	}

	for _, tr := range pro.GetTaskRuns() {
		stepMaterials, err := FromStepImages(tr.Status.Steps)
		if err != nil {
			return mats, err
		}
		mats = append(mats, stepMaterials...)

		// add sidecar images
		sidecarMaterials, err := FromSidecarImages(tr.Status.Sidecars)
		if err != nil {
			return nil, err
		}
		mats = append(mats, sidecarMaterials...)

		// add remote task configsource information in materials
		if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
			m := common.ProvenanceMaterial{
				URI:    tr.Status.Provenance.RefSource.URI,
				Digest: tr.Status.Provenance.RefSource.Digest,
			}
			mats = append(mats, m)
		}
	}

	mats = append(mats, FromPipelineParamsAndResults(ctx, pro)...)

	// remove duplicate materials
	mats, err := removeDuplicateMaterials(mats)
	if err != nil {
		return mats, err
	}
	return mats, nil
}

// FromStepImages gets predicate.materials from step images
func FromStepImages(steps []v1beta1.StepState) ([]common.ProvenanceMaterial, error) {
	mats := []common.ProvenanceMaterial{}
	for _, stepState := range steps {
		m, err := fromImageID(stepState.ImageID)
		if err != nil {
			return nil, err
		}
		mats = append(mats, m)
	}
	return mats, nil
}

// FromSidecarImages gets predicate.materials from sidecar images
func FromSidecarImages(sidecars []v1beta1.SidecarState) ([]common.ProvenanceMaterial, error) {
	mats := []common.ProvenanceMaterial{}
	for _, sidecarState := range sidecars {
		m, err := fromImageID(sidecarState.ImageID)
		if err != nil {
			return nil, err
		}
		mats = append(mats, m)
	}
	return mats, nil
}

// fromImageID converts an imageId with format <uri>@sha256:<digest> and generates a provenance materials.
func fromImageID(imageID string) (common.ProvenanceMaterial, error) {
	uriDigest := strings.Split(imageID, uriSeparator)
	if len(uriDigest) != 2 {
		return common.ProvenanceMaterial{}, fmt.Errorf("expected imageID %s to be separable by @", imageID)
	}
	digest := strings.Split(uriDigest[1], digestSeparator)
	if len(digest) != 2 {
		return common.ProvenanceMaterial{}, fmt.Errorf("expected imageID %s to be separable by @ and :", imageID)
	}
	uri := strings.TrimPrefix(uriDigest[0], "docker-pullable://")
	m := common.ProvenanceMaterial{
		Digest: common.DigestSet{},
	}
	m.URI = artifacts.OCIScheme + uri
	m.Digest[digest[0]] = digest[1]
	return m, nil
}

// FromTaskResourcesToMaterials gets materials from task resources.
func FromTaskResources(ctx context.Context, tro *objects.TaskRunObject) []common.ProvenanceMaterial {
	mats := []common.ProvenanceMaterial{}
	if tro.Spec.Resources != nil { //nolint:all //incompatible with pipelines v0.45
		// check for a Git PipelineResource
		for _, input := range tro.Spec.Resources.Inputs { //nolint:all //incompatible with pipelines v0.45
			if input.ResourceSpec == nil || input.ResourceSpec.Type != backport.PipelineResourceTypeGit { //nolint:all //incompatible with pipelines v0.45
				continue
			}

			m := common.ProvenanceMaterial{
				Digest: common.DigestSet{},
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
	}
	return mats
}

// FromTaskParamsAndResults scans over the taskrun, taskspec params and taskrun results
// and looks for unstructured type hinted names matching CHAINS-GIT_COMMIT and CHAINS-GIT_URL
// to extract the commit and url value for input artifact materials.
func FromTaskParamsAndResults(ctx context.Context, tro *objects.TaskRunObject) []common.ProvenanceMaterial {
	var commit, url string
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

	var mats []common.ProvenanceMaterial
	if commit != "" && url != "" {
		mats = append(mats, common.ProvenanceMaterial{
			URI: url,
			// TODO. this could be sha256 as well. Fix in another PR.
			Digest: map[string]string{"sha1": commit},
		})
	}

	sms := artifacts.RetrieveMaterialsFromStructuredResults(ctx, tro, artifacts.ArtifactsInputsResultName)
	mats = append(mats, sms...)

	return mats
}

// removeDuplicateMaterials removes duplicate materials from the slice of materials.
// Original order of materials is retained.
func removeDuplicateMaterials(mats []common.ProvenanceMaterial) ([]common.ProvenanceMaterial, error) {
	out := make([]common.ProvenanceMaterial, 0, len(mats))

	// make map to store seen materials
	seen := map[string]bool{}
	for _, mat := range mats {
		m, err := json.Marshal(mat)
		if err != nil {
			return nil, err
		}
		if seen[string(m)] {
			continue
		}

		seen[string(m)] = true
		out = append(out, mat)
	}
	return out, nil
}

// FromPipelineParamsAndResults extracts type hinted params and results and adds the url and digest to materials.
func FromPipelineParamsAndResults(ctx context.Context, pro *objects.PipelineRunObject) []common.ProvenanceMaterial {
	mats := []common.ProvenanceMaterial{}
	sms := artifacts.RetrieveMaterialsFromStructuredResults(ctx, pro, artifacts.ArtifactsInputsResultName)
	mats = append(mats, sms...)

	var commit, url string
	// search status.PipelineSpec.params
	if pro.Status.PipelineSpec != nil {
		for _, p := range pro.Status.PipelineSpec.Params {
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

	// search pipelineRunSpec.params
	for _, p := range pro.Spec.Params {
		if p.Name == attest.CommitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == attest.URLParam {
			url = p.Value.StringVal
		}
	}

	// search status.PipelineRunResults
	for _, r := range pro.Status.PipelineResults {
		if r.Name == attest.CommitParam {
			commit = r.Value.StringVal
		}
		if r.Name == attest.URLParam {
			url = r.Value.StringVal
		}
	}
	if len(commit) > 0 && len(url) > 0 {
		url = attest.SPDXGit(url, "")
		mats = append(mats, common.ProvenanceMaterial{
			URI:    url,
			Digest: map[string]string{"sha1": commit},
		})
	}
	return mats
}
