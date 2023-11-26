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
	"encoding/json"

	signing "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1/taskrun"
	v1beta1taskrunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/taskrun"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

const (
	// SecretPath contains the path to the secrets volume that is mounted in.
	SecretPath = "/etc/signing-secrets"
)

type ReconcilerV1 struct {
	TaskRunSigner     signing.Signer
	Pipelineclientset versioned.Interface
}

// Check that our Reconciler implements taskrunreconciler.Interface and taskrunreconciler.Finalizer
var _ taskrunreconciler.Interface = (*ReconcilerV1)(nil)
var _ taskrunreconciler.Finalizer = (*ReconcilerV1)(nil)

// ReconcileKind  handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *ReconcilerV1) ReconcileKind(ctx context.Context, tr *v1.TaskRun) pkgreconciler.Event {
	return r.FinalizeKind(ctx, tr)
}

// FinalizeKind implements taskrunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every taskrun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *ReconcilerV1) FinalizeKind(ctx context.Context, tr *v1.TaskRun) pkgreconciler.Event {
	cfg := *config.FromContext(ctx)

	// Check to see if chains is configured to watch v1beta1 Tekton API objects
	if cfg.TektonAPI.WatchForTektonV1Beta1APIInstead {
		return nil
	}

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

type ReconcilerV1Beta1 struct {
	TaskRunSigner     signing.Signer
	Pipelineclientset versioned.Interface
}

// Check that our Reconciler implements taskrunreconciler.Interface and taskrunreconciler.Finalizer
var _ v1beta1taskrunreconciler.Interface = (*ReconcilerV1Beta1)(nil)
var _ v1beta1taskrunreconciler.Finalizer = (*ReconcilerV1Beta1)(nil)

// ReconcileKind  handles a changed or created TaskRun.
// This is the main entrypoint for chains business logic.
func (r *ReconcilerV1Beta1) ReconcileKind(ctx context.Context, tr *v1beta1.TaskRun) pkgreconciler.Event {
	return r.FinalizeKind(ctx, tr)
}

// FinalizeKind implements taskrunreconciler.Finalizer
// We utilize finalizers to ensure that we get a crack at signing every taskrun
// that we see flowing through the system.  If we don't add a finalizer, it could
// get cleaned up before we see the final state and sign it.
func (r *ReconcilerV1Beta1) FinalizeKind(ctx context.Context, trV1Beta1 *v1beta1.TaskRun) pkgreconciler.Event {
	cfg := *config.FromContext(ctx)

	// Check to see if chains is configured to watch v1beta1 Tekton API objects
	if !cfg.TektonAPI.WatchForTektonV1Beta1APIInstead {
		return nil
	}

	// Check to make sure the TaskRun is finished.
	if !trV1Beta1.IsDone() {
		logging.FromContext(ctx).Infof("taskrun %s/%s is still running", trV1Beta1.Namespace, trV1Beta1.Name)
		return nil
	}

	trV1 := &v1.TaskRun{}
	err := trV1Beta1.ConvertTo(ctx, trV1)
	if err != nil {
		return err
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

	obj := objects.NewTaskRunObjectV1(trV1)
	objv1beta1 := objects.NewTaskRunObjectV1Beta1(trV1Beta1)

	// Check to see if it has already been signed.
	if signing.Reconciled(ctx, r.Pipelineclientset, objv1beta1) {
		logging.FromContext(ctx).Infof("taskrun %s/%s has been reconciled", trV1Beta1.Namespace, trV1Beta1.Name)
		return nil
	}

	if err := r.TaskRunSigner.Sign(ctx, obj, objv1beta1); err != nil {
		return err
	}

	return nil
}
