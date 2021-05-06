// +build e2e

/*
Copyright 2021 The Tekton Authors
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

package test

import (
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	taskName = "kaniko-task"
)

func kanikoTaskRun(namespace string) *v1beta1.TaskRun {
	return &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kaniko-taskrun",
			Namespace:    namespace,
		},
		Spec: v1beta1.TaskRunSpec{
			TaskRef: &v1beta1.TaskRef{
				Name: taskName,
			},
		},
	}
}

func kanikoTask(t *testing.T, namespace, destinationImage string) *v1beta1.Task {
	ref, err := name.ParseReference(destinationImage)
	if err != nil {
		t.Fatalf("unable to parse image name: %v", err)
	}
	return &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      taskName,
			Namespace: namespace,
		},
		Spec: v1beta1.TaskSpec{
			Results: []v1beta1.TaskResult{
				{Name: "IMAGE_URL"},
				{Name: "IMAGE_DIGEST"},
			},
			Steps: []v1beta1.Step{
				{
					Container: v1.Container{
						Name:  "create-dockerfile",
						Image: "bash:latest",
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "dockerfile",
								MountPath: "/dockerfile",
							},
						},
					},
					Script: "#!/usr/bin/env bash\necho \"FROM gcr.io/distroless/base@sha256:6ec6da1888b18dd971802c2a58a76a7702902b4c9c1be28f38e75e871cedc2df\" > /dockerfile/Dockerfile",
				}, {
					Container: v1.Container{
						Name:    "build-and-push",
						Image:   "gcr.io/kaniko-project/executor:v1.6.0",
						Command: []string{"/kaniko/executor"},
						Args: []string{
							"--dockerfile=/dockerfile/Dockerfile",
							fmt.Sprintf("--destination=%s", destinationImage),
							"--context=/dockerfile",
							"--digest-file=$(results.IMAGE_DIGEST.path)",
							// Need this to push the image to the insecure registry
							"--insecure",
						},
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "dockerfile",
								MountPath: "/dockerfile",
							},
						},
					},
				}, {
					Container: v1.Container{
						Name:  "save-image-url",
						Image: "bash:latest",
						VolumeMounts: []v1.VolumeMount{
							{
								Name:      "dockerfile",
								MountPath: "/dockerfile",
							},
						},
					},
					Script: fmt.Sprintf("#!/usr/bin/env bash\necho %s | tee $(results.IMAGE_URL.path)", ref.String()),
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "dockerfile",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
		},
	}
}
