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
	"log"

	chainsconfig "github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/server"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	ctx := context.Background()
	if err := setup(ctx); err != nil {
		log.Fatal(err)
	}
}

// setup sets up a configmap watcher and reconciles the server based on config values
func setup(ctx context.Context) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	// creates the clientset
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	watcher, err := client.CoreV1().ConfigMaps("tekton-chains").Watch(ctx, metav1.ListOptions{
		FieldSelector: fields.Set{"metadata.name": "chains-config"}.AsSelector().String(),
		Watch:         true,
	})
	if err != nil {
		return err
	}

	for event := range watcher.ResultChan() {
		log.Printf("Configmap watcher received event: %v", event.Type)
		cm := event.Object.(*v1.ConfigMap)
		cfg, err := chainsconfig.NewConfigFromConfigMap(cm)
		if err != nil {
			log.Printf("error generating new config: %v", err)
		}
		if err := server.Reconcile(ctx, *cfg); err != nil {
			log.Printf("error starting server: %v", err)
		}
	}
	return nil
}
