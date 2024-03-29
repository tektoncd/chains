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

package fake

import (
	"context"

	"github.com/tektoncd/chains/pkg/taskrunmetrics"
	_ "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun/fake" // Make sure the fake taskrun informer is setup
	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
)

func init() {
	injection.Fake.RegisterClient(func(ctx context.Context, _ *rest.Config) context.Context { return taskrunmetrics.WithClient(ctx) })
}
