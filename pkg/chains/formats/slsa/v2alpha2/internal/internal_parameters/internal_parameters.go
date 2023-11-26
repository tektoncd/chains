/*
Copyright 2023 The Tekton Authors

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

package internalparameters

import (
	"github.com/tektoncd/chains/pkg/chains/objects"
)

// SLSAInternalParameters provides the chains config as internalparameters
func SLSAInternalParameters(tko objects.TektonObject) map[string]any {
	internalParams := make(map[string]any)
	if provenance := tko.GetProvenance(); !provenance.IsNil() && !provenance.FeatureFlagsIsNil() {
		internalParams["tekton-pipelines-feature-flags"] = *provenance.GetFeatureFlags()
	}
	return internalParams
}

// TektonInternalParameters provides the chains config as well as annotations and labels
func TektonInternalParameters(tko objects.TektonObject) map[string]any {
	internalParams := make(map[string]any)
	if provenance := tko.GetProvenance(); !provenance.IsNil() && !provenance.FeatureFlagsIsNil() {
		internalParams["tekton-pipelines-feature-flags"] = *provenance.GetFeatureFlags()
	}
	internalParams["labels"] = tko.GetLabels()
	internalParams["annotations"] = tko.GetAnnotations()
	return internalParams
}
