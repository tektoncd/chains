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
	"flag"

	"github.com/tektoncd/chains/pkg/reconciler/pipelinerun"
	"github.com/tektoncd/chains/pkg/reconciler/taskrun"

	"k8s.io/client-go/rest"

	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"

	// Run with all of the upstream providers.
	// We link this here to give downstreams greater choice/control over
	// which providers they pull in, by linking their own variants in their
	// own binary entrypoint.
	_ "github.com/sigstore/cosign/v2/pkg/providers/all"

	// Register the provider-specific plugins
	_ "github.com/sigstore/sigstore/pkg/signature/kms/aws"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/azure"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/gcp"
	_ "github.com/sigstore/sigstore/pkg/signature/kms/hashivault"
)

func main() {
	flag.IntVar(&controller.DefaultThreadsPerController, "threads-per-controller", controller.DefaultThreadsPerController, "Threads (goroutines) to create per controller")
	namespace := flag.String("namespace", "", "Namespace to restrict informer to. Optional, defaults to all namespaces.")

	// This also calls flag.Parse().
	cfg := injection.ParseAndGetRESTConfigOrDie()

	if cfg.QPS == 0 {
		cfg.QPS = 2 * rest.DefaultQPS
	}
	if cfg.Burst == 0 {
		cfg.Burst = rest.DefaultBurst
	}

	flag.Parse()
	ctx := injection.WithNamespaceScope(signals.NewContext(), *namespace)

	sharedmain.MainWithConfig(ctx, "watcher", cfg, taskrun.NewController, pipelinerun.NewController)
}
