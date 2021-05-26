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
	"regexp"
	"sort"
	"strings"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/google/go-containerregistry/pkg/name"
)

const (
	tektonID           = "https://tekton.dev/attestations/chains@v1"
	commitParam        = "CHAINS-GIT_COMMIT"
	urlParam           = "CHAINS-GIT_URL"
	ociDigestResult    = "IMAGE_DIGEST"
	chainsDigestSuffix = "_DIGEST"
)

type artifactType int

const (
	artifactTypeUnknown artifactType = iota
	artifactTypeOCI
)

type InTotoIte6 struct {
	builderID string
}

type artifactResult struct {
	ParamName  string
	ResultName string
	Type       artifactType
}

func NewFormatter(cfg config.Config) (formats.Payloader, error) {
	return &InTotoIte6{builderID: cfg.Builder.ID}, nil
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
			PredicateType: intoto.PredicateProvenanceV01,
		},
		Predicate: intoto.ProvenancePredicate{
			Builder: intoto.ProvenanceBuilder{
				ID: i.builderID,
			},
			Recipe: intoto.ProvenanceRecipe{
				Type:       tektonID,
				EntryPoint: name,
			},
		},
	}
	if tr.Status.StartTime != nil {
		att.Predicate.Metadata.BuildStartedOn = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		att.Predicate.Metadata.BuildFinishedOn = &tr.Status.CompletionTime.Time
	}

	results := getResultDigests(tr)
	att.StatementHeader.Subject = getSubjectDigests(tr, results)

	definedInMaterial, mats := materials(tr)
	att.Predicate.Materials = mats
	att.Predicate.Recipe.DefinedInMaterial = definedInMaterial

	return att, nil
}

// materials will add the following to the attestation materials:
// 	1. All step containers
//  2. Any specification for git
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
			definedInMaterial = intP(i)
			break
		}
	}
	return definedInMaterial, m
}

func (i *InTotoIte6) Type() formats.PayloadType {
	return formats.PayloadTypeInTotoIte6
}

// getResultDigests scans over the task.Status.TaskSpec.Results to find any
// result that has a name that ends with _DIGEST.
func getResultDigests(tr *v1beta1.TaskRun) []artifactResult {
	results := []artifactResult{}
	// Scan for digests
	for _, r := range tr.Status.TaskRunResults {
		if strings.HasSuffix(r.Name, chainsDigestSuffix) {
			// 7 chars in _DIGEST
			at := artifactTypeUnknown

			// IMAGE_DIGEST always refers to an OCI image
			if r.Name == ociDigestResult {
				at = artifactTypeOCI
			}

			results = append(results, artifactResult{
				ParamName:  r.Name[:len(r.Name)-7],
				ResultName: r.Name,
				Type:       at,
			})
		}
	}

	return results
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

// getSubjectDigests uses depends on taskResults with names ending with
// _DIGEST.
// To be able to find the resource that matches the digest, it relies on a
// naming schema for an input parameter.
// If the output result from this task is foo_DIGEST, it expects to find an
// parameter named 'foo', and resolves this value to use for the subject with
// for the found digest.
// If no parameter is found, we search the results of this task, which allows
// for a task to create a random resource not known prior to execution that
// gets checksummed and added as the subject.
// Digests can be on two formats: $alg:$digest (commonly used for container
// image hashes), or $alg:$digest $path, which is used when a step is
// calculating a hash of a previous step.
func getSubjectDigests(tr *v1beta1.TaskRun, results []artifactResult) []intoto.Subject {
	subs := []intoto.Subject{}
	r := regexp.MustCompile(`(\S+)\s+(\S+)`)

	// Resolve *_DIGEST variables
Results:
	for _, ar := range results {
		// Look for subject amongst the params
		sub := ""
		for _, p := range tr.Spec.Params {
			if ar.ParamName == p.Name {
				if p.Value.Type != v1beta1.ParamTypeString {
					continue
				}
				sub = p.Value.StringVal
				break
			}
		}
		if sub == "" {
			// Loog amongst task results
			for _, trr := range tr.Status.TaskRunResults {
				if trr.Name == ar.ParamName {
					sub = strings.TrimRight(trr.Value, "\n")
					break
				}
			}
		}

		if sub == "" {
			// Parameter was not specifed for this task run
			continue
		}

		// Loop over the parameters
		for _, trr := range tr.Status.TaskRunResults {
			if trr.Name == ar.ResultName {
				// Value should be on format:
				// alg:hash
				// alg:hash filename
				mod := strings.TrimRight(trr.Value, "\n")
				ah := strings.Split(mod, ":")
				if len(ah) != 2 {
					continue
				}
				alg := ah[0]

				// Is format "hash filename"
				groups := r.FindStringSubmatch(ah[1])
				var hash string
				if len(groups) == 3 {
					hash = groups[1]
				} else {
					hash = ah[1]
				}

				// OCI image shall use pacakge url format for subjects
				if ar.Type == artifactTypeOCI {
					imageID := getOCIImageID(sub, alg, hash)
					sub, _, _ = getPackageURLDocker(imageID)
				}

				subs = append(subs, intoto.Subject{
					Name: sub,
					Digest: map[string]string{
						alg: hash,
					},
				})
				// Subject found, go after next result
				continue Results
			}
		}
	}

	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Name <= subs[j].Name
	})
	return subs
}

func intP(i int) *int {
	return &i
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
