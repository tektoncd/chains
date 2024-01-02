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
	"encoding/json"
	"fmt"

	"github.com/in-toto/in-toto-golang/in_toto"
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
	formats.RegisterPayloader(PayloadTypeInTotoIte6, NewPayloader)
	formats.RegisterPayloader(PayloadTypeSlsav1, NewPayloader)
}

type InTotoIte6 struct {
	slsaConfig *slsaconfig.SlsaConfig
}

func NewPayloader(cfg config.Config) (formats.Payloader, error) {
	return NewPayloaderFromConfig(cfg), nil
}

func NewPayloaderFromConfig(cfg config.Config) *InTotoIte6 {
	opts := []Option{
		WithBuilderID(cfg.Builder.ID),
		WithDeepInspection(cfg.Artifacts.PipelineRuns.DeepInspectionEnabled),
	}
	return NewFormatter(opts...)
}

type options struct {
	builderID      string
	deepInspection bool
}

type Option func(*options)

func WithDeepInspection(enabled bool) Option {
	return func(o *options) {
		o.deepInspection = enabled
	}
}

func WithBuilderID(id string) Option {
	return func(o *options) {
		o.builderID = id
	}
}

func NewFormatter(opts ...Option) *InTotoIte6 {
	o := &options{}
	for _, f := range opts {
		f(o)
	}

	return &InTotoIte6{
		slsaConfig: &slsaconfig.SlsaConfig{
			BuilderID:             o.builderID,
			DeepInspectionEnabled: o.deepInspection,
		},
	}
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
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (i *InTotoIte6) FormatPayload(ctx context.Context, obj objects.TektonObject) (*ProvenanceStatement, error) {
	var (
		s   *in_toto.ProvenanceStatement
		err error
	)

	switch v := obj.(type) {
	case *objects.TaskRunObject:
		s, err = taskrun.GenerateAttestation(ctx, v, i.slsaConfig)
	case *objects.PipelineRunObject:
		s, err = pipelinerun.GenerateAttestation(ctx, v, i.slsaConfig)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}

	if err != nil {
		return nil, err
	}
	// Wrap output in BinaryMarshaller so we know how to format this.
	out := ProvenanceStatement(*s)
	return &out, nil

}

func (i *InTotoIte6) Type() config.PayloadType {
	return formats.PayloadTypeSlsav1
}

type ProvenanceStatement in_toto.ProvenanceStatement

func (s ProvenanceStatement) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}
