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

package tekton

import (
	"context"
	"fmt"
	"testing"

	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	pipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func CreateObject(t *testing.T, ctx context.Context, ps pipelineclientset.Interface, obj objects.TektonObject) objects.TektonObject {
	switch o := obj.GetObject().(type) {
	case *v1.PipelineRun:
		pr, err := ps.TektonV1().PipelineRuns(obj.GetNamespace()).Create(ctx, o, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("error creating pipelinerun: %v", err)
		}
		return objects.NewPipelineRunObjectV1(pr)
	case *v1.TaskRun:
		tr, err := ps.TektonV1().TaskRuns(obj.GetNamespace()).Create(ctx, o, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("error creating taskrun: %v", err)
		}
		return objects.NewTaskRunObjectV1(tr)
	}
	return nil
}

// Passing in TektonObject since it encapsulates namespace, name, and type.
func GetObject(t *testing.T, ctx context.Context, ps pipelineclientset.Interface, obj objects.TektonObject) (objects.TektonObject, error) {
	switch obj.GetObject().(type) {
	case *v1.PipelineRun:
		return GetPipelineRun(t, ctx, ps, obj.GetNamespace(), obj.GetName())
	case *v1.TaskRun:
		return GetTaskRun(t, ctx, ps, obj.GetNamespace(), obj.GetName())
	}
	t.Fatalf("unknown object type %T", obj.GetObject())
	return nil, fmt.Errorf("unknown object type %T", obj.GetObject())
}

func GetPipelineRun(t *testing.T, ctx context.Context, ps pipelineclientset.Interface, namespace, name string) (objects.TektonObject, error) {
	pr, err := ps.TektonV1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting pipelinerun: %v", err)
	}
	return objects.NewPipelineRunObjectV1(pr), nil
}

func GetTaskRun(t *testing.T, ctx context.Context, ps pipelineclientset.Interface, namespace, name string) (objects.TektonObject, error) {
	tr, err := ps.TektonV1().TaskRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting taskrun: %v", err)
	}
	return objects.NewTaskRunObjectV1(tr), nil
}

func WatchObject(t *testing.T, ctx context.Context, ps pipelineclientset.Interface, obj objects.TektonObject) (watch.Interface, error) {
	switch o := obj.GetObject().(type) {
	case *v1.PipelineRun:
		return ps.TektonV1().PipelineRuns(obj.GetNamespace()).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
			Name:      o.GetName(),
			Namespace: o.GetNamespace(),
		}))
	case *v1.TaskRun:
		return ps.TektonV1().TaskRuns(obj.GetNamespace()).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
			Name:      o.GetName(),
			Namespace: o.GetNamespace(),
		}))
	}
	return nil, fmt.Errorf("unknown object type %T", obj.GetObject())
}
