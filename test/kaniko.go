//go:build e2e
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
	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	taskName = "kaniko-task"
)

func kanikoPipelineRun(ns string) objects.TektonObject {
	imagePipelineRun := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "image-pipelinerun",
			Namespace:    ns,
			Annotations:  map[string]string{chains.RekorAnnotation: "true"},
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "kaniko",
						TaskRef: &v1beta1.TaskRef{
							Name: "kaniko-task",
							Kind: v1beta1.NamespacedTaskKind,
						},
					},
				},
				Results: []v1beta1.PipelineResult{
					{
						Name:  "IMAGE_URL",
						Value: *v1beta1.NewArrayOrString("$(tasks.kaniko.results.IMAGE_URL)"),
					},
					{
						Name:  "IMAGE_DIGEST",
						Value: *v1beta1.NewArrayOrString("$(tasks.kaniko.results.IMAGE_DIGEST)"),
					},
				},
			},
		},
	}
	return objects.NewPipelineRunObject(&imagePipelineRun)
}

func kanikoTaskRun(namespace string) objects.TektonObject {
	tr := &v1beta1.TaskRun{
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
	return objects.NewTaskRunObject(tr)
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
					Name:  "create-dockerfile",
					Image: "bash:latest",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "dockerfile",
							MountPath: "/dockerfile",
						},
					},
					Script: "#!/usr/bin/env bash\necho \"FROM gcr.io/distroless/base@sha256:6ec6da1888b18dd971802c2a58a76a7702902b4c9c1be28f38e75e871cedc2df\" > /dockerfile/Dockerfile",
				}, {
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
				}, {
					Name:  "save-image-url",
					Image: "bash:latest",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "dockerfile",
							MountPath: "/dockerfile",
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

func verifyKanikoTaskRun(namespace, destinationImage, publicKey string) objects.TektonObject {
	script := `#!/busybox/sh
	
# save the public key
echo "%s" > cosign.pub

# verify the image
cosign verify --allow-insecure-registry --key cosign.pub %s

# verify the attestation
cosign verify-attestation --allow-insecure-registry --key cosign.pub %s`
	script = fmt.Sprintf(script, publicKey, destinationImage, destinationImage)

	return objects.NewTaskRunObject(&v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "verify-kaniko-taskrun",
			Namespace:    namespace,
		},
		Spec: v1beta1.TaskRunSpec{
			TaskSpec: &v1beta1.TaskSpec{
				Steps: []v1beta1.Step{
					{
						Name:   "verify-image",
						Image:  "gcr.io/projectsigstore/cosign/ci/cosign:d764e8b89934dc1043bd1b13112a66641c63a038@sha256:228c37f9f37415efbd6a4ff16aae81197206ce1410a227bcab8ac8b039b36237",
						Script: script,
					},
				},
			},
		},
	})
}
