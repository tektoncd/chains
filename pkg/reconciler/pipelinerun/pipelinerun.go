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

package pipelinerun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/pipelinerun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"

	pipelineRunFinalizerName = "chains.tekton.dev/pipelinerun"
)

type Reconciler struct {
	PipelineRunSigner signing.Signer
	Pipelineclientset versioned.Interface
	TaskRunLister     listers.TaskRunLister
	Tracker           tracker.Interface
}

// Check that our Reconciler implements pipelinerunreconciler.Interface and pipelinerunreconciler.Finalizer
var _ pipelinerunreconciler.Interface = (*Reconciler)(nil)
var _ pipelinerunreconciler.Finalizer = (*Reconciler)(nil)

// ReconcileKind  handles a changed or created PipelineRun.
// This is the main entrypoint for chains business logic.
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1.PipelineRun) pkgreconciler.Event {
	log := logging.FromContext(ctx).With("pipelinerun", fmt.Sprintf("%s/%s", pr.Namespace, pr.Name))
	return r.FinalizeKind(logging.WithLogger(ctx, log), pr)
}

// FinalizeKind implements pipelinerunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every pipelinerun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *Reconciler) FinalizeKind(ctx context.Context, pr *v1.PipelineRun) (result pkgreconciler.Event) { //nolint:ireturn
	// MIGRATION: When the framework dispatches DoFinalizeKind (DeletionTimestamp is set) and
	// finalization succeeds (returns nil), check if the finalizer was added via merge patch by the
	// old controller version. SSA cannot remove finalizers it doesn't own, so we remove it via
	// merge patch ourselves. This can be removed once all pre-SSA resources are deleted.
	//
	// The DeletionTimestamp guard is necessary because in Chains, ReconcileKind delegates to
	// FinalizeKind, so FinalizeKind is called both from the DoReconcileKind path (no
	// DeletionTimestamp) and the DoFinalizeKind path (DeletionTimestamp set). We must only
	// remove the finalizer when the object is actually being deleted.
	if !pr.DeletionTimestamp.IsZero() {
		defer func() {
			if result != nil {
				return
			}
			if !r.isFinalizerOwnedByMergePatch(pr) {
				return
			}
			logging.FromContext(ctx).Infof("Removing merge-patch finalizer on %s/%s for SSA migration",
				pr.Namespace, pr.Name)
			if err := r.removeFinalizerViaMergePatch(ctx, pr); err != nil {
				logging.FromContext(ctx).Warnw("Failed to remove finalizer via merge patch",
					zap.Error(err))
				result = controller.NewRequeueAfter(10 * time.Second)
			}
		}()
	}

	// Check to make sure the PipelineRun is finished.
	if !pr.IsDone() {
		logging.FromContext(ctx).Infof("pipelinerun is still running")
		return nil
	}
	pro := objects.NewPipelineRunObjectV1(pr)

	// Check to see if it has already been signed.
	if annotations.Reconciled(ctx, r.Pipelineclientset, pro) {
		logging.FromContext(ctx).Infof("pipelinerun has been reconciled")
		return nil
	}

	// Get TaskRun names depending on whether embeddedstatus feature is set or not
	var trs []string
	for _, cr := range pr.Status.ChildReferences {
		trs = append(trs, cr.Name)
	}

	// Signing both taskruns and pipelineruns causes a race condition when using oci storage
	// during the push to the registry. This checks the taskruns to ensure they've been reconciled
	// before attempting to sign the pippelinerun.
	for _, name := range trs {
		tr, err := r.TaskRunLister.TaskRuns(pr.Namespace).Get(name)
		if err != nil {
			logging.FromContext(ctx).Errorf("Unable to get reconciled status of taskrun %s within pipelinerun", name)
			if errors.IsNotFound(err) {
				// Since this is an unrecoverable scenario, returning the error would prevent the
				// finalizer from being removed, thus preventing the PipelineRun from being deleted.
				return nil
			}
			return err
		}
		if tr == nil {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not found", name)
			return nil
		}
		if tr.Status.CompletionTime == nil {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not yet finalized: status is not complete", name)
			return r.trackTaskRun(tr, pr)
		}
		reconciled := annotations.Reconciled(ctx, r.Pipelineclientset, objects.NewTaskRunObjectV1(tr))
		if !reconciled {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not yet reconciled", name)
			return r.trackTaskRun(tr, pr)
		}
		pro.AppendTaskRun(tr)
	}

	if err := r.PipelineRunSigner.Sign(ctx, pro); err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) trackTaskRun(tr *v1.TaskRun, pr *v1.PipelineRun) error {
	ref := tracker.Reference{
		APIVersion: "tekton.dev/v1",
		Kind:       "TaskRun",
		Namespace:  tr.Namespace,
		Name:       tr.Name,
	}
	return r.Tracker.TrackReference(ref, pr)
}

// isFinalizerOwnedByMergePatch checks if the finalizer was added via merge patch (Update operation).
// MIGRATION: This is a temporary migration feature to handle the upgrade scenario where
// in-flight PipelineRuns have finalizers set via merge patch by the old controller version.
// Kubernetes SSA treats (manager, Update) and (manager, Apply) as different owners, so we need
// to detect and handle the old ownership pattern.
// This function can be removed once all resources from the pre-SSA version are deleted.
func (r *Reconciler) isFinalizerOwnedByMergePatch(pr *v1.PipelineRun) bool {
	for _, mf := range pr.ManagedFields {
		if mf.Operation == metav1.ManagedFieldsOperationUpdate {
			raw := mf.FieldsV1.GetRawBytes()
			if raw != nil {
				if bytes.Contains(raw, []byte(`"f:finalizers"`)) &&
					bytes.Contains(raw, []byte(`v:\"chains.tekton.dev/pipelinerun\"`)) {
					return true
				}
			}
		}
	}
	return false
}

// removeFinalizerViaMergePatch removes the finalizer using merge patch.
// MIGRATION: This is a temporary migration feature to handle the upgrade scenario where
// in-flight PipelineRuns have finalizers set via merge patch by the old controller version.
// This uses merge patch to remove finalizers that cannot be removed via SSA due to different
// ownership (manager, Update) vs (manager, Apply).
// This function can be removed once all resources from the pre-SSA version are deleted.
func (r *Reconciler) removeFinalizerViaMergePatch(ctx context.Context, pr *v1.PipelineRun) error {
	var newFinalizers []string
	for _, f := range pr.Finalizers {
		if f != pipelineRunFinalizerName {
			newFinalizers = append(newFinalizers, f)
		}
	}

	mergePatch := map[string]interface{}{
		"metadata": map[string]interface{}{
			"finalizers":      newFinalizers,
			"resourceVersion": pr.ResourceVersion,
		},
	}

	patch, err := json.Marshal(mergePatch)
	if err != nil {
		return err
	}

	_, err = r.Pipelineclientset.TektonV1().PipelineRuns(pr.Namespace).Patch(
		ctx, pr.Name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}
