// +build e2e

/*
Copyright 2019 The Tekton Authors

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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstall(t *testing.T) {
	c, _ := setup(t)
	dep, err := c.KubeClient.AppsV1().Deployments("tekton-pipelines").Get("tekton-chains-controller", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Error getting chains deployment: %v", err)
	}
	if dep.Status.AvailableReplicas != 1 {
		t.Errorf("Chains installation not running correctly: %v", dep)
	}
}
