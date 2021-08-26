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

package storage

import (
	"reflect"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestInitializeBackends(t *testing.T) {

	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{{
		name: "none",
		want: []string{},
	}, {
		name: "tekton",
		want: []string{"tekton"},
		cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: "tekton"}}},
	}}
	logger := logtesting.TestLogger(t)
	ctx, _ := rtesting.SetupFakeContext(t)
	ps := fakepipelineclient.Get(ctx)
	kc := fakekubeclient.Get(ctx)
	tr := &v1beta1.TaskRun{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitializeBackends(ps, kc, logger, tr, tt.cfg)
			if err != nil {
				t.Errorf("InitializeBackends() error = %v", err)
				return
			}
			gotTypes := []string{}
			for _, g := range got {
				gotTypes = append(gotTypes, g.Type())
			}
			if !reflect.DeepEqual(gotTypes, tt.want) {
				t.Errorf("InitializeBackends() = %v, want %v", gotTypes, tt.want)
			}
		})
	}
}
