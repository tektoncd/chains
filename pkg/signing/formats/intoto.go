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
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// InToto is a formatter that translates a TaskRun to an InToto link file
type InToto struct {
}

// CreatePayload implements the Payloader interface.
func (i *InToto) CreatePayload(tr *v1beta1.TaskRun) (interface{}, error) {
	// Here we translate a Tekton TaskRun into an InToto Link file.
	// At a high leevel, the  mapping looks roughly like:
	// Input Resource Results -> Materials
	// Output Resource Results -> Products
	// The entire TaskRun body -> Environment

	l := in_toto.Link{
		Type: "_link",
	}

	// Populate materials with resource inputs.
	l.Materials = map[string]interface{}{}
	if tr.Spec.Resources != nil {
		for _, r := range tr.Spec.Resources.Inputs {
			for _, rr := range tr.Status.ResourcesResult {
				if r.Name == rr.ResourceName {
					if _, ok := l.Materials[rr.ResourceName]; !ok {
						l.Materials[rr.ResourceName] = map[string]string{}
					}
					m := l.Materials[rr.ResourceName].(map[string]string)
					m[rr.Key] = rr.Value
				}
			}
		}

		// Populate products with resource outputs.
		l.Products = map[string]interface{}{}
		for _, r := range tr.Spec.Resources.Outputs {
			for _, rr := range tr.Status.ResourcesResult {
				if r.Name == rr.ResourceName {
					if _, ok := l.Products[rr.ResourceName]; !ok {
						l.Products[rr.ResourceName] = map[string]string{}
					}
					m := l.Products[rr.ResourceName].(map[string]string)
					m[rr.Key] = rr.Value
				}
			}
		}
	}

	// The environment should contain info about Tekton itself,
	// and the container images used in the TaskRun.
	l.Environment = map[string]interface{}{}
	l.Environment["tekton"] = tr.Status

	// TODO: Add Tekton release info here
	return l, nil
}

func (i *InToto) Type() string {
	return PayloadTypeInToto
}
