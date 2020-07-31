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

package controller

import (
	"context"

	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/chains/pkg/signing"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	informers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
)

// Reconciler implements knative.dev/pkg/controller.Reconciler
type Reconciler struct {
	Logger            *zap.SugaredLogger
	TaskRunLister     informers.TaskRunLister
	TaskRunSigner     signing.Signer
	KubeClientSet     kubernetes.Interface
	PipelineClientSet versioned.Interface
	ConfigStore       *config.ConfigStore
}

// handleTaskRun handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) handleTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error {
	// Check to make sure the TaskRun is finished.
	if !tr.IsDone() {
		r.Logger.Infof("taskrun %s/%s is still running", tr.Namespace, tr.Name)
		return nil
	}
	// Check to see if it has already been signed.
	if signing.IsSigned(tr) {
		r.Logger.Infof("taskrun %s/%s has already been signed", tr.Namespace, tr.Name)
		return nil
	}

	if err := r.TaskRunSigner.SignTaskRun(tr); err != nil {
		return err
	}
	return nil
}

// Reconcile is the main entrypoint called when a Task is created or changed
func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	// Figure out the namespace and name from the key.
	r.Logger.Infof("reconciling resource key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.Logger.Errorf("invalid resource key: %s", key)
		return nil
	}
	// Get the TaskRun resource with this namespace/name
	tr, err := r.TaskRunLister.TaskRuns(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		r.Logger.Infof("task run %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		r.Logger.Errorf("Error retrieving TaskRun %q: %s", name, err)
		return err
	}

	r.Logger.Infof("Sending update for %s/%s (uid %s)", namespace, name, tr.UID)

	// Call the actual handler with a copy of the fetched TaskRun.
	return r.handleTaskRun(ctx, tr.DeepCopy())
}
