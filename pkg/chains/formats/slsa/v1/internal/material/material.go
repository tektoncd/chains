/*
Copyright 2022 The Tekton Authors

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

package material

import (
	"encoding/json"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

// RemoveDuplicateMaterials removes duplicate materials from the slice of materials
func RemoveDuplicateMaterials(mats []slsa.ProvenanceMaterial) ([]slsa.ProvenanceMaterial, error) {
	// make materialMap to store unique materials
	materialMap := map[string]slsa.ProvenanceMaterial{}
	for _, mat := range mats {
		m, err := json.Marshal(mat)
		if err != nil {
			return nil, err
		}
		materialMap[string(m)] = mat
	}
	distinctMaterials := make([]slsa.ProvenanceMaterial, 0, len(materialMap))
	for _, mat := range materialMap {
		distinctMaterials = append(distinctMaterials, mat)
	}
	return distinctMaterials, nil
}
