/*
Copyright 2024 The Tekton Authors
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

package results

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/objects"

	slsa "github.com/in-toto/attestation/go/v1"
)

var imageResultsNamesSuffixs = []string{
	artifacts.OCIImageURLResultName,
	artifacts.OCIImageDigestResultName,
}

// GetResultsWithoutBuildArtifacts returns all the results without those that are build artifacts.
func GetResultsWithoutBuildArtifacts(results []objects.Result, resultTypePrefix string) ([]*slsa.ResourceDescriptor, error) {
	byProd := []*slsa.ResourceDescriptor{}
	for _, r := range results {
		if isBuildArtifact, err := artifacts.IsBuildArtifact(r); err != nil || isBuildArtifact {
			continue
		}

		if isOCIImage(r.Name) {
			continue
		}

		content, err := json.Marshal(r.Value)
		if err != nil {
			return nil, err
		}

		byProd = append(byProd, &slsa.ResourceDescriptor{
			Name:      fmt.Sprintf(resultTypePrefix, r.Name),
			Content:   content,
			MediaType: "application/json",
		})
	}

	return byProd, nil
}

func isOCIImage(resName string) bool {
	for _, suffix := range imageResultsNamesSuffixs {
		if strings.HasSuffix(resName, suffix) {
			return true
		}
	}

	return resName == artifacts.OCIImagesResultName
}
