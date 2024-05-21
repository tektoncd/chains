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
	"fmt"

	buildtypes "github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/build_types"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

// SLSAInternalParameters provides the chains config as internalparameters
func SLSAInternalParameters(tko objects.TektonObject) map[string]any {
	internalParams := make(map[string]any)
	if provenance := tko.GetProvenance(); provenance != (*v1.Provenance)(nil) && provenance.FeatureFlags != nil {
		internalParams["tekton-pipelines-feature-flags"] = *provenance.FeatureFlags
	}
	return internalParams
}

// TektonInternalParameters provides the chains config as well as annotations and labels
func TektonInternalParameters(tko objects.TektonObject) map[string]any {
	internalParams := make(map[string]any)
	if provenance := tko.GetProvenance(); provenance != (*v1.Provenance)(nil) && provenance.FeatureFlags != nil {
		internalParams["tekton-pipelines-feature-flags"] = *provenance.FeatureFlags
	}
	internalParams["labels"] = tko.GetLabels()
	internalParams["annotations"] = tko.GetAnnotations()
	return internalParams
}

// GetInternalParamters returns the internal parameters for the given tekton object based on the build type.
func GetInternalParamters(obj objects.TektonObject, buildDefinitionType string) (map[string]any, error) {
	var internalParameters map[string]any

	switch buildDefinitionType {
	case buildtypes.SlsaBuildType:
		internalParameters = SLSAInternalParameters(obj)
	case buildtypes.TektonBuildType:
		internalParameters = TektonInternalParameters(obj)
	default:
		return nil, fmt.Errorf("unsupported buildType %v", buildDefinitionType)
	}

	return internalParameters, nil
}
