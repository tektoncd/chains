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
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/taskrunmetrics"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/taskrun"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	_ "github.com/tektoncd/chains/pkg/chains/formats/all"
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)
	taskRunInformer := taskruninformer.Get(ctx)

	kubeClient := kubeclient.Get(ctx)
	pipelineClient := pipelineclient.Get(ctx)

	tsSigner := &chains.ObjectSigner{
		SecretPath:        SecretPath,
		Pipelineclientset: pipelineClient,
		Recorder:          taskrunmetrics.Get(ctx),
	}

	c := &Reconciler{
		TaskRunSigner:     tsSigner,
		Pipelineclientset: pipelineClient,
	}
	impl := taskrunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
		watcherStop := make(chan bool)

		cfgStore := config.NewConfigStore(logger, func(name string, value interface{}) {
			select {
			case watcherStop <- true:
				logger.Info("sent close event to WatchBackends()...")
			default:
				logger.Info("could not send close event to WatchBackends()...")
			}

			// get updated config
			cfg := *value.(*config.Config)

			// get all backends for storing provenance
			backends, err := storage.InitializeBackends(ctx, pipelineClient, kubeClient, cfg)
			if err != nil {
				logger.Error(err)
			}
			tsSigner.Backends = backends

			if err := storage.WatchBackends(ctx, watcherStop, tsSigner.Backends, cfg); err != nil {
				logger.Error(err)
			}
		})

		// setup watches for the config names provided by client
		cfgStore.WatchConfigs(cmw)

		return controller.Options{
			// The chains reconciler shouldn't mutate the taskrun's status.
			SkipStatusUpdates: true,
			ConfigStore:       cfgStore,
			FinalizerName:     "chains.tekton.dev",
		}
	})

	if _, err := taskRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue)); err != nil {
		logger.Errorf("adding event handler for taskrun controller's taskrun informer encountered error: %w", err)
	}

	return impl
}
