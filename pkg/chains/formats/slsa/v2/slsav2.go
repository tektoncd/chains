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

package v2

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/v2/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
)

const (
	PayloadTypeSlsav2 = formats.PayloadTypeSlsav2
)

func init() {
	formats.RegisterPayloader(PayloadTypeSlsav2, NewFormatter)
}

type Slsa struct {
	builderID string
}

func NewFormatter(cfg config.Config) (formats.Payloader, error) {
	return &Slsa{
		builderID: cfg.Builder.ID,
	}, nil
}

func (s *Slsa) Wrap() bool {
	return true
}

func (s *Slsa) CreatePayload(ctx context.Context, obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case *objects.TaskRunObject:
		return taskrun.GenerateAttestation(ctx, s.builderID, s.Type(), v)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (s *Slsa) Type() config.PayloadType {
	return formats.PayloadTypeSlsav2
}
