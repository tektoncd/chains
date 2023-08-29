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
package compare

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v1"
)

// SLSAV1CompareOptions returns the comparison options for sorting some slice fields in
// SLSA v1 statement including ResourceDescriptor and Subject.
func SLSAV1CompareOptions() []cmp.Option {
	// checking content + uri + digest should be sufficient here based on the fact that
	// a ResourceDescriptor MUST specify one of uri, digest or content at a minimum.
	// Source: https://github.com/in-toto/attestation/blob/main/spec/v1/resource_descriptor.md#fields
	resourceDescriptorSort := func(x, y slsa.ResourceDescriptor) bool {
		if string(x.Content) != string(y.Content) {
			return string(x.Content) < string(y.Content)
		}
		if x.URI != y.URI {
			return x.URI < y.URI
		}
		return lessDigestSet(x.Digest, y.Digest)
	}

	return []cmp.Option{
		cmpopts.SortSlices(resourceDescriptorSort),
		SubjectCompareOption(),
	}
}

// SubjectCompareOption returns the comparison option to sort and compare a
// list of Subjects.
func SubjectCompareOption() cmp.Option {
	subjectSort := func(x, y in_toto.Subject) bool {
		if x.Name != y.Name {
			return x.Name < y.Name
		}
		return lessDigestSet(x.Digest, y.Digest)
	}
	return cmpopts.SortSlices(subjectSort)
}

// MaterialsCompareOption returns the comparison option to sort and compare a
// list of Materials.
func MaterialsCompareOption() cmp.Option {
	materialsSort := func(x, y common.ProvenanceMaterial) bool {
		if x.URI != y.URI {
			return x.URI < y.URI
		}
		return lessDigestSet(x.Digest, y.Digest)
	}
	return cmpopts.SortSlices(materialsSort)
}

func lessDigestSet(x, y common.DigestSet) bool {
	for algo, digestX := range x {
		digestY, ok := y[algo]
		if !ok {
			// Algorithm not present in y, x is considered greater.
			return false
		}
		// Compare the digests lexicographically.
		if digestX != digestY {
			return digestX < digestY
		}
		// The digests are equal, check the next algorithm.
	}

	// All algorithms in x have corresponding entries in y, so check if y has more algorithms.
	return len(x) < len(y)
}
