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
	"github.com/tektoncd/chains/pkg/chains/formats/sbom"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v1/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
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
	appContext   context.Context
	builderID    string
	sbomMaxBytes int64
	slsaConfig   *slsaconfig.SlsaConfig
}

func NewFormatter(ctx context.Context, cfg config.Config) (formats.Payloader, error) {
	return &InTotoIte6{
		appContext:   ctx,
		builderID:    cfg.Builder.ID,
		sbomMaxBytes: cfg.Artifacts.SBOM.MaxBytes,
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
	case *objects.TaskRunObject:
		return taskrun.GenerateAttestation(ctx, v, i.slsaConfig)
	case *objects.PipelineRunObject:
		return pipelinerun.GenerateAttestation(ctx, v, i.slsaConfig)
	case *objects.SBOMObject:
		// TODO: It is odd that the slsa package has a dependency on the sbom package. But,
		// this is required for now since intotoite6 is currently a part of the slsa package.
		kc := kubeclient.Get(i.appContext)
		// TODO: Use SLSAConfig (or some variation of it? SBOMConfig?)
		return sbom.GenerateAttestation(ctx, kc, i.builderID, i.sbomMaxBytes, v)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (i *InTotoIte6) Type() config.PayloadType {
	return formats.PayloadTypeSlsav1
}
