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

package pipelinerun

import (
	"context"

	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/storage"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/pipelinerunmetrics"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/pipelinerun"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1/taskrun"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/pipelinerun"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"

	_ "github.com/tektoncd/chains/pkg/chains/formats/all"
)

func NewController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)
	pipelineRunInformer := pipelineruninformer.Get(ctx)
	taskRunInformer := taskruninformer.Get(ctx)

	kubeClient := kubeclient.Get(ctx)
	pipelineClient := pipelineclient.Get(ctx)

	psSigner := &chains.ObjectSigner{
		SecretPath:        SecretPath,
		Pipelineclientset: pipelineClient,
		Recorder:          pipelinerunmetrics.Get(ctx),
	}

	c := &Reconciler{
		PipelineRunSigner: psSigner,
		Pipelineclientset: pipelineClient,
		TaskRunLister:     taskRunInformer.Lister(),
	}

	impl := pipelinerunreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
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
			psSigner.Backends = backends

			if err := storage.WatchBackends(ctx, watcherStop, psSigner.Backends, cfg); err != nil {
				logger.Error(err)
			}
		})

		// setup watches for the config names provided by client
		cfgStore.WatchConfigs(cmw)

		return controller.Options{
			// The chains reconciler shouldn't mutate the pipelinerun's status.
			SkipStatusUpdates: true,
			ConfigStore:       cfgStore,
			FinalizerName:     "chains.tekton.dev/pipelinerun", // TODO: unique name required?
		}
	})

	c.Tracker = impl.Tracker

	if _, err := pipelineRunInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue)); err != nil {
		logger.Errorf("adding event handler for pipelinerun controller's pipelinerun informer encountered error: %w", err)
	}

	if _, err := taskRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1.PipelineRun{}),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	}); err != nil {
		logger.Errorf("adding event handler for pipelinerun controller's taskrun informer encountered error: %w", err)
	}

	return impl
}
