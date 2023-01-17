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

package test

import (
	"github.com/google/go-cmp/cmp/cmpopts"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

// OptSortMaterial provides compare options to sort a slice of materials by URI in the provenance
var OptSortMaterial = cmpopts.SortSlices(func(i, j slsa.ProvenanceMaterial) bool {
	if i.URI == j.URI {
		i_DigestValue := ""
		for _, v := range i.Digest {
			i_DigestValue = v
		}
		j_DigestValue := ""
		for _, v := range j.Digest {
			j_DigestValue = v
		}
		return i_DigestValue < j_DigestValue
	}
	return i.URI < j.URI
})
