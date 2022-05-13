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
	"sort"
	"testing"

	"github.com/tektoncd/chains/pkg/config"
	fakepipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client/fake"
	"k8s.io/apimachinery/pkg/util/sets"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
)

func TestInitializeBackends(t *testing.T) {

	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{
			name: "none",
			want: []string{},
		},
		{
			name: "tekton",
			want: []string{"tekton"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("tekton")}}},
		},
		// TODO: Re-enable this test when it doesn't rely on ambient GCP credentials.
		//{
		//	name: "gcs",
		//	want: []string{"gcs"},
		//	cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("gcs")}}},
		//},
		{
			name: "oci",
			want: []string{"oci"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("oci")}}},
		},
		// TODO: Re-enable this test when it doesn't rely on ambient GCP credentials.
		// {
		// 	name: "grafeas",
		// 	want: []string{"grafeas"},
		// 	cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("grafeas")}}},
		// },
		{
			name: "multi",
			want: []string{"oci", "tekton"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("oci", "tekton")}}},
		},
		{
			name: "pubsub",
			want: []string{"pubsub"},
			cfg:  config.Config{Artifacts: config.ArtifactConfigs{TaskRuns: config.Artifact{StorageBackend: sets.NewString("pubsub")}}}},
	}
	logger := logtesting.TestLogger(t)
	ctx, _ := rtesting.SetupFakeContext(t)
	ps := fakepipelineclient.Get(ctx)
	kc := fakekubeclient.Get(ctx)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitializeBackends(ctx, ps, kc, logger, tt.cfg)
			if err != nil {
				t.Errorf("InitializeBackends() error = %v", err)
				return
			}
			logger.Debugf("Backend: %v", got)
			gotTypes := []string{}
			for _, g := range got {
				gotTypes = append(gotTypes, g.Type())
			}
			sort.Strings(gotTypes)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(gotTypes, tt.want) {
				t.Errorf("InitializeBackends() = %v, want %v", gotTypes, tt.want)
			}
		})
	}
}
