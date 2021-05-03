/*
Copyright 2020 The Tekton Authors
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

package artifacts

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

type Signable interface {
	ExtractObjects(tr *v1beta1.TaskRun) []interface{}
	StorageBackend(cfg config.Config) string
	Signer(cfg config.Config) string
	PayloadFormat(cfg config.Config) formats.PayloadType
	Key(interface{}) string
	Type() string
}

type TaskRunArtifact struct {
	Logger *zap.SugaredLogger
}

func (ta *TaskRunArtifact) Key(obj interface{}) string {
	tr := obj.(*v1beta1.TaskRun)
	return "taskrun-" + string(tr.UID)
}

func (ta *TaskRunArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	return []interface{}{tr}
}
func (ta *TaskRunArtifact) Type() string {
	return "tekton"
}

func (ta *TaskRunArtifact) StorageBackend(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.StorageBackend
}

func (ta *TaskRunArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.TaskRuns.Format)
}

func (ta *TaskRunArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.Signer
}

type OCIArtifact struct {
	Logger *zap.SugaredLogger
}

type image struct {
	url    string
	digest string
}

func (oa *OCIArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	imageResourceNames := map[string]*image{}
	if tr.Status.TaskSpec != nil && tr.Status.TaskSpec.Resources != nil {
		for _, output := range tr.Status.TaskSpec.Resources.Outputs {
			if output.Type == v1beta1.PipelineResourceTypeImage {
				imageResourceNames[output.Name] = &image{}
			}
		}
	}

	for _, rr := range tr.Status.ResourcesResult {
		img, ok := imageResourceNames[rr.ResourceName]
		if !ok {
			continue
		}
		// We have a result for an image!
		if rr.Key == "url" {
			img.url = rr.Value
		} else if rr.Key == "digest" {
			img.digest = rr.Value
		}
	}

	objs := []interface{}{}
	for _, image := range imageResourceNames {
		dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", image.url, image.digest))
		if err != nil {
			oa.Logger.Error(err)
			continue
		}
		objs = append(objs, dgst)
	}

	// Now check TaskResults
	taskResultImage := image{}
	for _, res := range tr.Status.TaskRunResults {
		if res.Name == "IMAGE_URL" {
			taskResultImage.url = strings.Trim(res.Value, "\n")
		} else if res.Name == "IMAGE_DIGEST" {
			taskResultImage.digest = strings.Trim(res.Value, "\n")
		}
	}
	// Only add it if we got both the URL and digest.
	if taskResultImage.url != "" && taskResultImage.digest != "" {
		dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", taskResultImage.url, taskResultImage.digest))
		if err != nil {
			oa.Logger.Error(err)
			return nil
		}
		objs = append(objs, dgst)
	}
	return objs
}

func (oa *OCIArtifact) Type() string {
	return "oci"
}

func (oa *OCIArtifact) StorageBackend(cfg config.Config) string {
	return cfg.Artifacts.OCI.StorageBackend
}

func (oa *OCIArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.OCI.Format)
}

func (oa *OCIArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.OCI.Signer
}

func (ta *OCIArtifact) Key(obj interface{}) string {
	v := obj.(name.Digest)
	return strings.TrimPrefix(v.DigestStr(), "sha256:")[:12]
}
