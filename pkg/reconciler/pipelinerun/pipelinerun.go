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
	"context"
	"encoding/json"
	"fmt"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/pipelinerun"
	v1beta1pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1"
	v1beta1listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
	"knative.dev/pkg/tracker"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
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
func (r *Reconciler) FinalizeKind(ctx context.Context, pr *v1.PipelineRun) pkgreconciler.Event {
	cfg := *config.FromContext(ctx)

	// Check to see if chains is configured to watch v1beta1 Tekton API objects
	if cfg.TektonAPI.WatchForTektonV1Beta1APIInstead {
		return nil
	}

	// Check to make sure the PipelineRun is finished.
	if !pr.IsDone() {
		logging.FromContext(ctx).Infof("pipelinerun is still running")
		return nil
	}
	pro := objects.NewPipelineRunObjectV1(pr)

	// Check to see if it has already been signed.
	if signing.Reconciled(ctx, r.Pipelineclientset, pro) {
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
		reconciled := signing.Reconciled(ctx, r.Pipelineclientset, objects.NewTaskRunObjectV1(tr))
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

type ReconcilerV1Beta1 struct {
	PipelineRunSigner signing.Signer
	Pipelineclientset versioned.Interface
	TaskRunLister     v1beta1listers.TaskRunLister
	Tracker           tracker.Interface
}

// Check that our Reconciler implements pipelinerunreconciler.Interface and pipelinerunreconciler.Finalizer
var _ v1beta1pipelinerunreconciler.Interface = (*ReconcilerV1Beta1)(nil)
var _ v1beta1pipelinerunreconciler.Finalizer = (*ReconcilerV1Beta1)(nil)

// ReconcileKind  handles a changed or created PipelineRun.
// This is the main entrypoint for chains business logic.
func (r *ReconcilerV1Beta1) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	log := logging.FromContext(ctx).With("pipelinerun", fmt.Sprintf("%s/%s", pr.Namespace, pr.Name))
	return r.FinalizeKind(logging.WithLogger(ctx, log), pr)
}

// FinalizeKind implements pipelinerunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every pipelinerun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *ReconcilerV1Beta1) FinalizeKind(ctx context.Context, prV1Beta1 *v1beta1.PipelineRun) pkgreconciler.Event {
	cfg := *config.FromContext(ctx)

	// Check to see if chains is configured to watch v1beta1 Tekton API objects
	if !cfg.TektonAPI.WatchForTektonV1Beta1APIInstead {
		return nil
	}

	// Check to make sure the PipelineRun is finished.
	if !prV1Beta1.IsDone() {
		logging.FromContext(ctx).Infof("pipelinerun is still running")
		return nil
	}

	prV1 := &v1.PipelineRun{}
	err := prV1Beta1.ConvertTo(ctx, prV1)
	if err != nil {
		return err
	}
	obj := objects.NewPipelineRunObjectV1(prV1)
	objv1beta1 := objects.NewPipelineRunObjectV1Beta1(prV1Beta1)

	// Check to see if it has already been signed.
	if signing.Reconciled(ctx, r.Pipelineclientset, objv1beta1) {
		logging.FromContext(ctx).Infof("pipelinerun has been reconciled")
		return nil
	}

	// Get TaskRun names depending on whether embeddedstatus feature is set or not
	var trs []string
	if len(prV1Beta1.Status.ChildReferences) == 0 || len(prV1Beta1.Status.TaskRuns) > 0 || len(prV1Beta1.Status.Runs) > 0 { //nolint:all //incompatible with pipelines v0.45
		for trName, ptrs := range prV1Beta1.Status.TaskRuns { //nolint:all //incompatible with pipelines v0.45
			// TaskRuns within a PipelineRun may not have been finalized yet if the PipelineRun timeout
			// has exceeded. Wait to process the PipelineRun on the next update, see
			// https://github.com/tektoncd/pipeline/issues/4916
			if ptrs.Status == nil || ptrs.Status.CompletionTime == nil {
				logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not yet finalized: embedded status is not complete", trName)
				return nil
			}
			trs = append(trs, trName)
		}
	} else {
		for _, cr := range prV1Beta1.Status.ChildReferences {
			trs = append(trs, cr.Name)
		}
	}

	// Signing both taskruns and pipelineruns causes a race condition when using oci storage
	// during the push to the registry. This checks the taskruns to ensure they've been reconciled
	// before attempting to sign the pippelinerun.
	for _, name := range trs {
		trV1Beta1, err := r.TaskRunLister.TaskRuns(prV1Beta1.Namespace).Get(name)
		if err != nil {
			logging.FromContext(ctx).Errorf("Unable to get reconciled status of taskrun %s within pipelinerun", name)
			if errors.IsNotFound(err) {
				// Since this is an unrecoverable scenario, returning the error would prevent the
				// finalizer from being removed, thus preventing the PipelineRun from being deleted.
				return nil
			}
			return err
		}

		trV1 := &v1.TaskRun{}
		err = trV1Beta1.ConvertTo(ctx, trV1)
		if err != nil {
			return err
		}

		if trV1 == nil {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not found", name)
			return nil
		}
		if trV1.Status.CompletionTime == nil {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not yet finalized: status is not complete", name)
			return r.trackTaskRun(trV1Beta1, prV1Beta1)
		}
		reconciled := signing.Reconciled(ctx, r.Pipelineclientset, objects.NewTaskRunObjectV1(trV1))
		if !reconciled {
			logging.FromContext(ctx).Infof("taskrun %s within pipelinerun is not yet reconciled", name)
			return r.trackTaskRun(trV1Beta1, prV1Beta1)
		}

		if trV1Beta1.Spec.Resources != nil {
			jsonData, err := json.Marshal(trV1Beta1.Spec.Resources)
			if err != nil {
				return err
			}
			trV1.Annotations["tekton.dev/v1beta1-spec-resources"] = string(jsonData)
		}

		if trV1Beta1.Status.TaskRunStatusFields.TaskSpec != nil && trV1Beta1.Status.TaskRunStatusFields.TaskSpec.Resources != nil {
			jsonData, err := json.Marshal(trV1Beta1.Status.TaskRunStatusFields.TaskSpec.Resources)
			if err != nil {
				return err
			}
			trV1.Annotations["tekton.dev/v1beta1-status-taskrunstatusfields-taskspec-resources"] = string(jsonData)
		}

		obj.AppendTaskRun(trV1)
	}

	if err := r.PipelineRunSigner.Sign(ctx, obj, objv1beta1); err != nil {
		return err
	}
	return nil
}

func (r *ReconcilerV1Beta1) trackTaskRun(tr *v1beta1.TaskRun, pr *v1beta1.PipelineRun) error {
	ref := tracker.Reference{
		APIVersion: "tekton.dev/v1beta1",
		Kind:       "TaskRun",
		Namespace:  tr.Namespace,
		Name:       tr.Name,
	}
	return r.Tracker.TrackReference(ref, pr)
}
