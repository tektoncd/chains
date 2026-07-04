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
	"bytes"
	"context"
	"encoding/json"
	"time"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/taskrun"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"

	taskRunFinalizerName    = "chains.tekton.dev/taskrun"
	oldTaskRunFinalizerName = "chains.tekton.dev"
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
func (r *Reconciler) FinalizeKind(ctx context.Context, tr *v1.TaskRun) (result pkgreconciler.Event) { //nolint:ireturn
	// MIGRATION: When the framework dispatches DoFinalizeKind (DeletionTimestamp is set) and
	// finalization succeeds (returns nil), check if the finalizer was added via merge patch by the
	// old controller version. SSA cannot remove finalizers it doesn't own, so we remove it via
	// merge patch ourselves. This can be removed once all pre-SSA resources are deleted.
	//
	// The DeletionTimestamp guard is necessary because in Chains, ReconcileKind delegates to
	// FinalizeKind, so FinalizeKind is called both from the DoReconcileKind path (no
	// DeletionTimestamp) and the DoFinalizeKind path (DeletionTimestamp set). We must only
	// remove the finalizer when the object is actually being deleted.
	if !tr.DeletionTimestamp.IsZero() {
		defer func() {
			if result != nil {
				return
			}
			if !r.isFinalizerOwnedByMergePatch(tr) {
				return
			}
			logging.FromContext(ctx).Infof("Removing merge-patch finalizer on %s/%s for SSA migration",
				tr.Namespace, tr.Name)
			if err := r.removeFinalizerViaMergePatch(ctx, tr); err != nil {
				logging.FromContext(ctx).Warnw("Failed to remove finalizer via merge patch",
					zap.Error(err))
				result = controller.NewRequeueAfter(10 * time.Second)
			}
		}()
	}

	// Check to make sure the TaskRun is finished.
	if !tr.IsDone() {
		logging.FromContext(ctx).Infof("taskrun %s/%s is still running", tr.Namespace, tr.Name)
		return nil
	}

	obj := objects.NewTaskRunObjectV1(tr)

	// Check to see if it has already been signed.
	if annotations.Reconciled(ctx, r.Pipelineclientset, obj) {
		logging.FromContext(ctx).Infof("taskrun %s/%s has been reconciled", tr.Namespace, tr.Name)
		return nil
	}

	if err := r.TaskRunSigner.Sign(ctx, obj); err != nil {
		return err
	}
	return nil
}

// isFinalizerOwnedByMergePatch checks if the finalizer was added via merge patch (Update operation).
// MIGRATION: This is a temporary migration feature to handle the upgrade scenario where
// in-flight TaskRuns have finalizers set via merge patch by the old controller version.
// Kubernetes SSA treats (manager, Update) and (manager, Apply) as different owners, so we need
// to detect and handle the old ownership pattern.
// This function can be removed once all resources from the pre-SSA version are deleted.
func (r *Reconciler) isFinalizerOwnedByMergePatch(tr *v1.TaskRun) bool {
	for _, mf := range tr.ManagedFields {
		if mf.Operation == metav1.ManagedFieldsOperationUpdate {
			raw := mf.FieldsV1.GetRawBytes()
			if raw != nil {
				if bytes.Contains(raw, []byte(`"f:finalizers"`)) &&
					(bytes.Contains(raw, []byte(`v:\"chains.tekton.dev/taskrun\"`)) ||
						bytes.Contains(raw, []byte(`v:\"chains.tekton.dev\"`))) {
					return true
				}
			}
		}
	}
	return false
}

// removeFinalizerViaMergePatch removes the finalizer using merge patch.
// MIGRATION: This is a temporary migration feature to handle the upgrade scenario where
// in-flight TaskRuns have finalizers set via merge patch by the old controller version.
// This uses merge patch to remove finalizers that cannot be removed via SSA due to different
// ownership (manager, Update) vs (manager, Apply).
// This function can be removed once all resources from the pre-SSA version are deleted.
func (r *Reconciler) removeFinalizerViaMergePatch(ctx context.Context, tr *v1.TaskRun) error {
	var newFinalizers []string
	for _, f := range tr.Finalizers {
		if f != taskRunFinalizerName && f != oldTaskRunFinalizerName {
			newFinalizers = append(newFinalizers, f)
		}
	}

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      newFinalizers,
			"resourceVersion": tr.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return err
	}

	_, err = r.Pipelineclientset.TektonV1().TaskRuns(tr.Namespace).Patch(
		ctx, tr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}
