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
	"fmt"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/artifact"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"knative.dev/pkg/logging"
)

const (
	uriSeparator    = "@"
	digestSeparator = ":"
)

// TaskMaterials constructs `predicate.materials` section by collecting all the artifacts that influence a taskrun such as source code repo and step&sidecar base images.
func TaskMaterials(ctx context.Context, tro *objects.TaskRunObjectV1) ([]common.ProvenanceMaterial, error) {
	var mats []common.ProvenanceMaterial

	// add step images
	stepMaterials, err := FromStepImages(tro)
	if err != nil {
		return nil, err
	}
	mats = artifact.AppendMaterials(mats, stepMaterials...)

	// add sidecar images
	sidecarMaterials, err := FromSidecarImages(tro)
	if err != nil {
		return nil, err
	}
	mats = artifact.AppendMaterials(mats, sidecarMaterials...)

	mats = artifact.AppendMaterials(mats, FromTaskParamsAndResults(ctx, tro)...)

	return mats, nil
}

func PipelineMaterials(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) ([]common.ProvenanceMaterial, error) {
	logger := logging.FromContext(ctx)
	var mats []common.ProvenanceMaterial
	if p := pro.Status.Provenance; p != nil && p.RefSource != nil {
		m := common.ProvenanceMaterial{
			URI:    p.RefSource.URI,
			Digest: p.RefSource.Digest,
		}
		mats = artifact.AppendMaterials(mats, m)
	}
	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			taskRuns := pro.GetTaskRunsFromTask(t.Name)
			if len(taskRuns) == 0 {
				logger.Infof("no taskruns found for task %s", t.Name)
				continue
			}
			for _, tr := range taskRuns {
				// Ignore Tasks that did not execute during the PipelineRun.
				if tr == nil || tr.Status.CompletionTime == nil {
					logger.Infof("taskrun status not found for task %s", t.Name)
					continue
				}
				stepMaterials, err := FromStepImages(tr)
				if err != nil {
					return mats, err
				}
				mats = artifact.AppendMaterials(mats, stepMaterials...)
				// add sidecar images
				sidecarMaterials, err := FromSidecarImages(tr)
				if err != nil {
					return nil, err
				}
				mats = artifact.AppendMaterials(mats, sidecarMaterials...)

				// add remote task configsource information in materials
				if tr.Status.Provenance != nil && tr.Status.Provenance.RefSource != nil {
					m := common.ProvenanceMaterial{
						URI:    tr.Status.Provenance.RefSource.URI,
						Digest: tr.Status.Provenance.RefSource.Digest,
					}
					mats = artifact.AppendMaterials(mats, m)
				}
			}
		}
	}

	mats = artifact.AppendMaterials(mats, FromPipelineParamsAndResults(ctx, pro, slsaconfig)...)

	return mats, nil
}

// FromStepImages gets predicate.materials from step images
func FromStepImages(tro *objects.TaskRunObjectV1) ([]common.ProvenanceMaterial, error) {
	mats := []common.ProvenanceMaterial{}
	for _, image := range tro.GetStepImages() {
		m, err := fromImageID(image)
		if err != nil {
			return nil, err
		}
		mats = artifact.AppendMaterials(mats, m)
	}
	return mats, nil
}

// FromSidecarImages gets predicate.materials from sidecar images
func FromSidecarImages(tro *objects.TaskRunObjectV1) ([]common.ProvenanceMaterial, error) {
	mats := []common.ProvenanceMaterial{}
	for _, image := range tro.GetSidecarImages() {
		m, err := fromImageID(image)
		if err != nil {
			return nil, err
		}
		mats = artifact.AppendMaterials(mats, m)
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

// FromTaskParamsAndResults scans over the taskrun, taskspec params and taskrun results
// and looks for unstructured type hinted names matching CHAINS-GIT_COMMIT and CHAINS-GIT_URL
// to extract the commit and url value for input artifact materials.
func FromTaskParamsAndResults(ctx context.Context, tro *objects.TaskRunObjectV1) []common.ProvenanceMaterial {
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

	for _, r := range tro.Status.Results {
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
		mats = artifact.AppendMaterials(mats, common.ProvenanceMaterial{
			URI: url,
			// TODO. this could be sha256 as well. Fix in another PR.
			Digest: map[string]string{"sha1": commit},
		})
	}

	sms := artifacts.RetrieveMaterialsFromStructuredResults(ctx, tro.GetResults())
	mats = artifact.AppendMaterials(mats, sms...)

	return mats
}

// FromStepActionsResults extracts type hinted results from StepActions associated with the TaskRun and adds the url and digest to materials.
func FromStepActionsResults(ctx context.Context, tro *objects.TaskRunObjectV1) (mats []common.ProvenanceMaterial) {
	for _, s := range tro.Status.Steps {
		var sCommit, sURL string
		for _, r := range s.Results {
			if r.Name == attest.CommitParam {
				sCommit = r.Value.StringVal
				continue
			}

			if r.Name == attest.URLParam {
				sURL = r.Value.StringVal
			}
		}

		sURL = attest.SPDXGit(sURL, "")
		if sCommit != "" && sURL != "" {
			mats = artifact.AppendMaterials(mats, common.ProvenanceMaterial{
				URI:    sURL,
				Digest: map[string]string{"sha1": sCommit},
			})
		}
	}
	sms := artifacts.RetrieveMaterialsFromStructuredResults(ctx, tro.GetStepResults())
	mats = artifact.AppendMaterials(mats, sms...)
	return
}

// FromPipelineParamsAndResults extracts type hinted params and results and adds the url and digest to materials.
func FromPipelineParamsAndResults(ctx context.Context, pro *objects.PipelineRunObjectV1, slsaconfig *slsaconfig.SlsaConfig) []common.ProvenanceMaterial {
	mats := []common.ProvenanceMaterial{}
	sms := artifacts.RetrieveMaterialsFromStructuredResults(ctx, pro.GetResults())
	mats = artifact.AppendMaterials(mats, sms...)

	var commit, url string

	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		// search type hinting param/results from each individual taskruns
		if slsaconfig.DeepInspectionEnabled {
			logger := logging.FromContext(ctx)
			pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
			for _, t := range pipelineTasks {
				taskRuns := pro.GetTaskRunsFromTask(t.Name)
				if len(taskRuns) == 0 {
					logger.Infof("no taskruns found for task %s", t.Name)
					continue
				}
				for _, tr := range taskRuns {
					// Ignore Tasks that did not execute during the PipelineRun.
					if tr == nil || tr.Status.CompletionTime == nil {
						logger.Infof("taskrun is not found or not completed for the task %s", t.Name)
						continue
					}
					materialsFromTasks := FromTaskParamsAndResults(ctx, tr)
					mats = artifact.AppendMaterials(mats, materialsFromTasks...)
				}
			}
		}

		// search status.PipelineSpec.params
		for _, p := range pSpec.Params {
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

	// search status.Results
	for _, r := range pro.Status.Results {
		if r.Name == attest.CommitParam {
			commit = r.Value.StringVal
		}
		if r.Name == attest.URLParam {
			url = r.Value.StringVal
		}
	}
	if len(commit) > 0 && len(url) > 0 {
		url = attest.SPDXGit(url, "")
		mats = artifact.AppendMaterials(mats, common.ProvenanceMaterial{
			URI:    url,
			Digest: map[string]string{"sha1": commit},
		})
	}
	return mats
}
