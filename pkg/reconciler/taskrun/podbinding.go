/*
Copyright 2026 The Tekton Authors
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
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/annotations"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

// ExecutionVerifiedAnnotation is set to "false" on a TaskRun when Chains could not
// confirm that the Pod named in its status ran it. Policy engines and admission
// controllers can use it to decide whether to trust the provenance.
const ExecutionVerifiedAnnotation = annotations.ChainsAnnotationPrefix + "execution-verified"

type podBinding int

const (
	podBound podBinding = iota
	// podMissing means the TaskRun names no Pod or the Pod is gone. Pods are often
	// cleaned up before Chains gets to a TaskRun, so this alone is not suspicious.
	podMissing
	// podMismatch means the Pod exists but is not controlled by this TaskRun, which
	// points at a forged or tampered TaskRun status.
	podMismatch
)

// verifyPodBinding runs the security.verify-pod-binding check and reports whether
// signing should go ahead. Runs that fail the check are annotated with
// ExecutionVerifiedAnnotation=false. In block mode a Pod that is not controlled by
// the TaskRun marks the run as failed instead of signing it.
func (r *Reconciler) verifyPodBinding(ctx context.Context, tr *v1.TaskRun, obj objects.TektonObject) (bool, error) {
	mode := config.FromContext(ctx).Security.VerifyPodBinding
	if mode == "" || mode == config.VerifyPodBindingOff || r.KubeClientSet == nil {
		return true, nil
	}
	logger := logging.FromContext(ctx)

	binding, reason, err := r.checkPodBinding(ctx, tr)
	if err != nil {
		return false, err
	}
	notVerified := map[string]string{ExecutionVerifiedAnnotation: "false"}
	switch binding {
	case podBound:
		return true, nil
	case podMissing:
		logger.Warnf("Could not verify the pod of taskrun %s/%s: %s", tr.Namespace, tr.Name, reason)
	case podMismatch:
		logger.Errorf("Pod binding check failed for taskrun %s/%s: %s", tr.Namespace, tr.Name, reason)
		if mode == config.VerifyPodBindingBlock {
			logger.Errorf("Not signing taskrun %s/%s because %s is %q", tr.Namespace, tr.Name, "security.verify-pod-binding", mode)
			return false, annotations.MarkFailed(ctx, obj, r.Pipelineclientset, notVerified)
		}
	}
	if err := annotations.AddAnnotations(ctx, obj, r.Pipelineclientset, notVerified); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) checkPodBinding(ctx context.Context, tr *v1.TaskRun) (podBinding, string, error) {
	podName := tr.Status.PodName
	if podName == "" {
		return podMissing, "the taskrun status names no pod", nil
	}
	pod, err := r.KubeClientSet.CoreV1().Pods(tr.Namespace).Get(ctx, podName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return podMissing, fmt.Sprintf("pod %s no longer exists", podName), nil
	}
	if err != nil {
		return podBound, "", fmt.Errorf("getting pod %s/%s: %w", tr.Namespace, podName, err)
	}
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.Kind != "TaskRun" || owner.Name != tr.Name || owner.UID != tr.UID {
		return podMismatch, fmt.Sprintf("pod %s is not controlled by taskrun %s (uid %s)", podName, tr.Name, tr.UID), nil
	}
	return podBound, "", nil
}
