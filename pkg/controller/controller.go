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

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	informers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
)

// Reconciler implements knative.dev/pkg/controller.Reconciler
type Reconciler struct {
	*reconciler.Base
	Logger        *zap.SugaredLogger
	TaskRunLister informers.TaskRunLister
}

// handleTaskRun handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) handleTaskRun(ctx context.Context, tr *v1beta1.TaskRun) error {
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

	// Call the actual handler with the fetched TaskRun.
	return r.handleTaskRun(ctx, tr)
}
