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

package taskrun

import (
	"context"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/taskrun"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
)

type Reconciler struct {
	TaskRunSigner     signing.Signer
	Pipelineclientset versioned.Interface
}

// Check that our Reconciler implements taskrunreconciler.Interface and taskrunreconciler.Finalizer
var _ taskrunreconciler.Interface = (*Reconciler)(nil)
var _ taskrunreconciler.Finalizer = (*Reconciler)(nil)

// ReconcileKind  handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) ReconcileKind(ctx context.Context, tr *v1.TaskRun) pkgreconciler.Event {
	return r.FinalizeKind(ctx, tr)
}

// FinalizeKind implements taskrunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every taskrun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *Reconciler) FinalizeKind(ctx context.Context, tr *v1.TaskRun) pkgreconciler.Event {
	// Check to make sure the TaskRun is finished.
	if !tr.IsDone() {
		logging.FromContext(ctx).Infof("taskrun %s/%s is still running", tr.Namespace, tr.Name)
		return nil
	}

	obj := objects.NewTaskRunObjectV1(tr)

	// Check to see if it has already been signed.
	if signing.Reconciled(ctx, r.Pipelineclientset, obj) {
		logging.FromContext(ctx).Infof("taskrun %s/%s has been reconciled", tr.Namespace, tr.Name)
		return nil
	}

	if err := r.TaskRunSigner.Sign(ctx, obj); err != nil {
		return err
	}
	return nil
}
