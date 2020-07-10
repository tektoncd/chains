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

package main

import (
	"context"
	"flag"

	tkcontroller "github.com/tektoncd/chains/pkg/controller"
	"github.com/tektoncd/chains/pkg/signing"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	"github.com/tektoncd/pipeline/pkg/reconciler"
	"k8s.io/client-go/tools/cache"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

var (
	namespace = flag.String("namespace", "", "Namespace to restrict informer to. Optional, defaults to all namespaces.")
)

const (
	controllerName = "chains"
)

func main() {
	flag.Parse()

	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), *namespace), "watcher",
		func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
			// TODO: store and use the cmw
			logger := logging.FromContext(ctx)
			taskRunInformer := taskruninformer.Get(ctx)
			kubeclientset := kubeclient.Get(ctx)
			pipelineclientset := pipelineclient.Get(ctx)

			c := &tkcontroller.Reconciler{
				Base: &reconciler.Base{
					KubeClientSet:     kubeclientset,
					PipelineClientSet: pipelineclientset,
				},
				Logger:        logger,
				TaskRunLister: taskRunInformer.Lister(),
				TaskRunSigner: &signing.TaskRunSigner{
					Pipelineclientset: pipelineclientset,
					Logger:            logger,
				},
			}
			impl := controller.NewImpl(c, c.Logger, controllerName)

			taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
				AddFunc:    impl.Enqueue,
				UpdateFunc: controller.PassNew(impl.Enqueue),
			})

			return impl
		})
}
