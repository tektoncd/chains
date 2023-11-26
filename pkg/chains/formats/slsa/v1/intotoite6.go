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

package v1

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
)

const (
	PayloadTypeInTotoIte6 = formats.PayloadTypeInTotoIte6
	PayloadTypeSlsav1     = formats.PayloadTypeSlsav1
)

func init() {
	formats.RegisterPayloader(PayloadTypeInTotoIte6, NewFormatter)
	formats.RegisterPayloader(PayloadTypeSlsav1, NewFormatter)
}

type InTotoIte6 struct {
	slsaConfig *slsaconfig.SlsaConfig
}

func NewFormatter(cfg config.Config) (formats.Payloader, error) {
	return &InTotoIte6{
		slsaConfig: &slsaconfig.SlsaConfig{
			BuilderID:             cfg.Builder.ID,
			DeepInspectionEnabled: cfg.Artifacts.PipelineRuns.DeepInspectionEnabled,
		},
	}, nil
}

func (i *InTotoIte6) Wrap() bool {
	return true
}

func (i *InTotoIte6) CreatePayload(ctx context.Context, obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case *objects.TaskRunObjectV1:
		return taskrun.GenerateAttestation(ctx, v, i.slsaConfig)
	case *objects.PipelineRunObjectV1:
		return pipelinerun.GenerateAttestation(ctx, v, i.slsaConfig)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (i *InTotoIte6) Type() config.PayloadType {
	return formats.PayloadTypeSlsav1
}
