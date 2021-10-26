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

package intotoite6

import (
	"fmt"
	"sort"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/provenance"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"

	"github.com/google/go-containerregistry/pkg/name"
)

const (
	tektonID           = "https://tekton.dev/attestations/chains@v1"
	commitParam        = "CHAINS-GIT_COMMIT"
	urlParam           = "CHAINS-GIT_URL"
	ociDigestResult    = "IMAGE_DIGEST"
	chainsDigestSuffix = "_DIGEST"
)

type InTotoIte6 struct {
	builderID string
	logger    *zap.SugaredLogger
}

func NewFormatter(cfg config.Config, logger *zap.SugaredLogger) (formats.Payloader, error) {
	return &InTotoIte6{
		builderID: cfg.Builder.ID,
		logger:    logger,
	}, nil
}

func (i *InTotoIte6) Wrap() bool {
	return true
}

func (i *InTotoIte6) CreatePayload(obj interface{}) (interface{}, error) {
	var tr *v1beta1.TaskRun
	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		tr = v
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
	return i.generateAttestationFromTaskRun(tr)
}

// generateAttestationFromTaskRun translates a Tekton TaskRun into an InToto ite6 provenance
// attestation.
// Spec: https://github.com/in-toto/attestation/blob/main/spec/predicates/provenance.md
// At a high level, the mapping looks roughly like:
// 	Configured builder id -> Builder.Id
// 	Results with name *_DIGEST -> Subject
// 	Step containers -> Materials
// 	Params with name CHAINS-GIT_* -> Materials and recipe.materials

// 	tekton-chains -> Recipe.type
// 	Taskname -> Recipe.entry_point
func (i *InTotoIte6) generateAttestationFromTaskRun(tr *v1beta1.TaskRun) (interface{}, error) {
	name := tr.Name
	if tr.Spec.TaskRef != nil {
		name = tr.Spec.TaskRef.Name
	}
	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: intoto.PredicateSLSAProvenanceV01,
		},
		Predicate: intoto.ProvenancePredicate{
			Metadata: &in_toto.ProvenanceMetadata{},
			Builder: intoto.ProvenanceBuilder{
				ID: i.builderID,
			},
			Recipe: intoto.ProvenanceRecipe{
				Type:       tektonID,
				EntryPoint: name,
				Arguments:  provenance.Steps(tr),
			},
		},
	}
	if tr.Status.StartTime != nil {
		att.Predicate.Metadata.BuildStartedOn = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		att.Predicate.Metadata.BuildFinishedOn = &tr.Status.CompletionTime.Time
	}

	att.StatementHeader.Subject = provenance.GetSubjectDigests(tr, i.logger)

	definedInMaterial, mats := materials(tr)
	att.Predicate.Materials = mats
	att.Predicate.Recipe.DefinedInMaterial = definedInMaterial

	return att, nil
}

// materials will add the following to the attestation materials:
// 1. All step containers
// 2. Any specification for git
func materials(tr *v1beta1.TaskRun) (*int, []intoto.ProvenanceMaterial) {
	var m []intoto.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tr)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		m = append(m, intoto.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"git_commit": gitCommit},
		})
	}
	// Add all found step containers as materials
	for _, trs := range tr.Status.Steps {
		uri, alg, digest := getPackageURLDocker(trs.ImageID)
		if uri == "" {
			continue
		}

		m = append(m, intoto.ProvenanceMaterial{
			URI:    uri,
			Digest: intoto.DigestSet{alg: digest},
		})
	}
	sort.Slice(m, func(i, j int) bool {
		return m[i].URI <= m[j].URI
	})
	// find the index of the Git material, if it exists, in the newly sorted list
	var definedInMaterial *int
	for i, u := range m {
		if u.URI == gitURL {
			definedInMaterial = pointer.Int(i)
			break
		}
	}
	return definedInMaterial, m
}

func (i *InTotoIte6) Type() formats.PayloadType {
	return formats.PayloadTypeInTotoIte6
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tr *v1beta1.TaskRun) (commit string, url string) {
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == commitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == urlParam {
			url = p.Value.StringVal
			// make sure url is PURL (git+https)
			if !strings.HasPrefix(url, "git+") {
				url = "git+" + url
			}
		}
	}
	return
}

// getPackageURLDocker takes an image id and creates a package URL string
// based from it.
// https://github.com/package-url/purl-spec
func getPackageURLDocker(imageID string) (string, string, string) {
	// Default registry per purl spec
	const defaultRegistry = "hub.docker.com"

	// imageID formats: name@alg:digest
	//                  schema://name@alg:digest
	d := strings.Split(imageID, "//")
	if len(d) == 2 {
		// Get away with schema
		imageID = d[1]
	}

	digest, err := name.NewDigest(imageID, name.WithDefaultRegistry(defaultRegistry))
	if err != nil {
		return "", "", ""
	}

	// DigestStr() is alg:digest
	parts := strings.Split(digest.DigestStr(), ":")
	if len(parts) != 2 {
		return "", "", ""
	}

	purl := fmt.Sprintf("pkg:docker/%s@%s",
		digest.Context().RepositoryStr(),
		digest.DigestStr(),
	)

	// Only inlude registry if it's not the default
	registry := digest.Context().Registry.Name()
	if registry != defaultRegistry {
		purl = fmt.Sprintf("%s?repository_url=%s", purl, registry)
	}

	return purl, parts[0], parts[1]
}

// getOCIImageID generates an imageID that is compatible imageID field in
// the task result's status field.
func getOCIImageID(name, alg, digest string) string {
	// image id is: docker://name@alg:digest
	return fmt.Sprintf("docker://%s@%s:%s", name, alg, digest)
}
