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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/taskrun"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
)

type Reconciler struct {
	TaskRunSigner signing.Signer
}

// Check that our Reconciler implements taskrunreconciler.Interface
var _ taskrunreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind  handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) ReconcileKind(ctx context.Context, tr *v1beta1.TaskRun) pkgreconciler.Event {
	// Check to make sure the TaskRun is finished.
	if !tr.IsDone() {
		logging.FromContext(ctx).Infof("taskrun %s/%s is still running", tr.Namespace, tr.Name)
		return nil
	}
	// Check to see if it has already been signed.
	if signing.IsSigned(tr) {
		logging.FromContext(ctx).Infof("taskrun %s/%s has already been signed", tr.Namespace, tr.Name)
		return nil
	}

	if err := r.TaskRunSigner.SignTaskRun(ctx, tr); err != nil {
		return err
	}
	return nil
}
