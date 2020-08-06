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
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/signing/formats"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
)

type Signable interface {
	ExtractObjects(tr *v1beta1.TaskRun) []interface{}
	StorageBackend(cfg config.Config) string
	PayloadFormat(cfg config.Config) formats.PayloadType
	Key(obj interface{}) string
	Type() string
}

type TaskRunArtifact struct {
	Logger *zap.SugaredLogger
}

func (ta *TaskRunArtifact) Key(obj interface{}) string {
	// Return something unique within the scope of the TaskRun.
	// In this case the taskrun is unique, so we don't need anything else.
	return "taskrun"
}

func (ta *TaskRunArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	return []interface{}{tr}
}
func (ta *TaskRunArtifact) Type() string {
	return "Tekton"
}

func (ta *TaskRunArtifact) StorageBackend(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.StorageBackend
}

func (ta *TaskRunArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.TaskRuns.Format)
}
