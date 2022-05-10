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

package formats

import (
	"context"
	"testing"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/spire"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestVerifySpire(t *testing.T) {
	spireMockClient := &spire.MockClient{}
	var (
		cc spire.ControllerAPIClient = spireMockClient
	)

	ctx := context.Background()

	testCases := []struct {
		// description of test
		desc string
		// function to tamper
		tamperCondition apis.Condition
		// annotations to set
		setAnnotations map[string]string
		// whether sign/verify procedure should succeed
		success bool
	}{
		{
			desc: "non-intrusive tamper with status annotation",
			tamperCondition: apis.Condition{
				Type:   apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
				Status: corev1.ConditionTrue,
			},
			setAnnotations: map[string]string{
				"unrelated-hash": "change",
			},
			success: true,
		},
		{
			desc: "tamper status hash annotation",
			tamperCondition: apis.Condition{
				Type:   apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
				Status: corev1.ConditionTrue,
			},
			setAnnotations: map[string]string{
				spire.TaskRunStatusHashAnnotation: "change-hash",
			},
			success: false,
		},
		{
			desc: "tamper condition fail",
			tamperCondition: apis.Condition{
				Type:   apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
				Status: corev1.ConditionFalse,
			},
			success: false,
		},
		{
			desc: "Spire not verified",
			tamperCondition: apis.Condition{
				Type:   apis.ConditionType(v1beta1.TaskRunConditionResultsVerified.String()),
				Status: corev1.ConditionTrue,
			},
			setAnnotations: map[string]string{
				spire.NotVerifiedAnnotation: "yes",
			},
			success: false,
		},
	}

	for _, tt := range testCases {
		for _, tr := range testTaskRuns() {

			success := func() bool {
				err := cc.AppendStatusInternalAnnotation(ctx, tr)
				if err != nil {
					return false
				}

				if tr.Status.Status.Annotations == nil {
					tr.Status.Status.Annotations = map[string]string{}
				}

				if tt.setAnnotations != nil {
					for k, v := range tt.setAnnotations {
						tr.Status.Status.Annotations[k] = v
					}
				}

				tr.Status.Status.Conditions = append(tr.Status.Status.Conditions, tt.tamperCondition)

				err = VerifySpire(ctx, tr, cc, logtesting.TestLogger(t))
				return err == nil
			}()

			if success != tt.success {
				t.Fatalf("test %v expected verify %v, got %v", tt.desc, tt.success, success)
			}
		}
	}
}

func objectMeta(name, ns string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   ns,
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}
}

func testTaskRuns() []*v1beta1.TaskRun {
	return []*v1beta1.TaskRun{
		// taskRun 1
		{
			ObjectMeta: objectMeta("taskrun-example", "foo"),
			Spec: v1beta1.TaskRunSpec{
				TaskRef: &v1beta1.TaskRef{
					Name:       "taskname",
					APIVersion: "a1",
				},
				ServiceAccountName: "test-sa",
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						apis.Condition{
							Type: apis.ConditionSucceeded,
						},
					},
				},
			},
		},
		// taskRun 2
		{
			ObjectMeta: objectMeta("taskrun-example-populated", "foo"),
			Spec: v1beta1.TaskRunSpec{
				TaskRef:            &v1beta1.TaskRef{Name: "unit-test-task"},
				ServiceAccountName: "test-sa",
				Resources:          &v1beta1.TaskRunResources{},
				Timeout:            &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						apis.Condition{
							Type: apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Steps: []v1beta1.StepState{{
						ContainerState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: int32(0)},
						},
					}},
				},
			},
		},
		// taskRun 3
		{
			ObjectMeta: objectMeta("taskrun-example-with-objmeta", "foo"),
			Spec: v1beta1.TaskRunSpec{
				TaskRef:            &v1beta1.TaskRef{Name: "unit-test-task"},
				ServiceAccountName: "test-sa",
				Resources:          &v1beta1.TaskRunResources{},
				Timeout:            &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						apis.Condition{
							Type: apis.ConditionSucceeded,
						},
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Steps: []v1beta1.StepState{{
						ContainerState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: int32(0)},
						},
					}},
				},
			},
		},
		// taskRun 4
		{
			ObjectMeta: objectMeta("taskrun-example-with-objmeta-annotations", "foo"),
			Spec: v1beta1.TaskRunSpec{
				TaskRef:            &v1beta1.TaskRef{Name: "unit-test-task"},
				ServiceAccountName: "test-sa",
				Resources:          &v1beta1.TaskRunResources{},
				Timeout:            &metav1.Duration{Duration: config.DefaultTimeoutMinutes * time.Minute},
			},
			Status: v1beta1.TaskRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						apis.Condition{
							Type: apis.ConditionSucceeded,
						},
					},
					Annotations: map[string]string{
						"annotation1": "a1value",
						"annotation2": "a2value",
					},
				},
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					Steps: []v1beta1.StepState{{
						ContainerState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{ExitCode: int32(0)},
						},
					}},
				},
			},
		},
	}
}
