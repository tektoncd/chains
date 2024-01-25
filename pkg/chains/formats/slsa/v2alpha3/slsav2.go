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

package v2alpha3

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha3/internal/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
)

const (
	PayloadTypeSlsav2alpha3 = formats.PayloadTypeSlsav2alpha3
)

func init() {
	formats.RegisterPayloader(PayloadTypeSlsav2alpha3, NewFormatter)
}

type Slsa struct {
	slsaConfig *slsaconfig.SlsaConfig
}

func NewFormatter(cfg config.Config) (formats.Payloader, error) {
	return &Slsa{
		slsaConfig: &slsaconfig.SlsaConfig{
			BuilderID:             cfg.Builder.ID,
			DeepInspectionEnabled: cfg.Artifacts.PipelineRuns.DeepInspectionEnabled,
			BuildType:             cfg.BuildDefinition.BuildType,
		},
	}, nil
}

func (s *Slsa) Wrap() bool {
	return true
}

func (s *Slsa) CreatePayload(ctx context.Context, obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case *objects.TaskRunObjectV1:
		return taskrun.GenerateAttestation(ctx, v, s.slsaConfig)
	case *objects.PipelineRunObjectV1:
		return pipelinerun.GenerateAttestation(ctx, v, s.slsaConfig)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (s *Slsa) Type() config.PayloadType {
	return formats.PayloadTypeSlsav2alpha3
}
