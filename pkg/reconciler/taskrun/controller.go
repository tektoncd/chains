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

package taskrun

import (
	"context"

	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/config"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/taskrun"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)
	taskRunInformer := taskruninformer.Get(ctx)

	c := &Reconciler{
		TaskRunSigner: &chains.TaskRunSigner{
			KubeClient:        kubeclient.Get(ctx),
			Pipelineclientset: pipelineclient.Get(ctx),
			SecretPath:        SecretPath,
		},
	}
	impl := taskrunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
		cfgStore := config.NewConfigStore(logger)
		cfgStore.WatchConfigs(cmw)

		return controller.Options{
			// The chains reconciler shouldn't mutate the taskrun's status.
			SkipStatusUpdates: true,
			ConfigStore:       cfgStore,
			FinalizerName:     "chains.tekton.dev",
		}
	})

	taskRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
