/*
Copyright 2024 The Tekton Authors

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

package v2alpha4

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha4/internal/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2alpha4/internal/taskrun"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
)

const (
	payloadTypeSlsav2alpha4 = formats.PayloadTypeSlsav2alpha4
)

func init() {
	formats.RegisterPayloader(payloadTypeSlsav2alpha4, NewFormatter)
}

// Slsa is a v2alpha4 payloader implementation.
type Slsa struct {
	slsaConfig *slsaconfig.SlsaConfig
}

// NewFormatter returns a new v2alpha4 payloader.
func NewFormatter(cfg config.Config) (formats.Payloader, error) { //nolint:ireturn
	return &Slsa{
		slsaConfig: &slsaconfig.SlsaConfig{
			BuilderID:             cfg.Builder.ID,
			DeepInspectionEnabled: cfg.Artifacts.PipelineRuns.DeepInspectionEnabled,
			BuildType:             cfg.BuildDefinition.BuildType,
		},
	}, nil
}

// Wrap indicates if the resulting payload should be wrapped.
func (s *Slsa) Wrap() bool {
	return true
}

// CreatePayload returns the payload for the given object using the v2alpha4 formatter logic.
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

// Type returns the version of this payloader.
func (s *Slsa) Type() config.PayloadType {
	return payloadTypeSlsav2alpha4
}
