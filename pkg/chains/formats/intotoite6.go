package formats

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	tektonID = "https://tekton.dev/attestations/chains@v1"
)

const (
	commitParam = "CHAINS-GIT_COMMIT"
	urlParam    = "CHAINS-GIT_URL"
)

type InTotoIte6 struct {
}

type artifactResult struct {
	ParamName  string
	ResultName string
}

type vcsInfo struct {
	Commit string
	URL    string
}

func (i *InTotoIte6) CreatePayload(obj interface{}) (interface{}, error) {
	var tr *v1beta1.TaskRun

	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		tr = v
	default:
		return nil, fmt.Errorf("unsupported type %s", v)

	}

	// Here we translate a Tekton TaskRun into an InToto ite6 attestation.
	// At a high leevel, the  mapping looks roughly like:
	// Podname -> Builder.Id
	// Results with name *-DIGEST -> Subject
	// Step containers -> Materials
	// Params with name CHAINS-GIT_* -> Materials and recipe.materials
	// tekton-chains -> Recipe.type
	// Taskname -> Recipe.entry_point
	att := in_toto.ProvenanceStatement{
		StatementHeader: in_toto.StatementHeader{
			Type:          in_toto.StatementInTotoV01,
			PredicateType: in_toto.PredicateProvenanceV01,
		},
		Predicate: in_toto.ProvenancePredicate{
			Builder: in_toto.ProvenanceBuilder{
				ID: tr.Status.PodName,
			},
			Recipe: in_toto.ProvenanceRecipe{
				Type:       tektonID,
				EntryPoint: tr.Spec.TaskRef.Name,
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

	att.Predicate.Materials = []in_toto.ProvenanceMaterial{}
	vcsInfo := getVcsInfo(tr)
	var recipeMaterialUri string

	// Store git rev as Materials and Recipe.Material
	if vcsInfo != nil {
		att.Predicate.Materials = append(att.Predicate.Materials, in_toto.ProvenanceMaterial{
			URI:    vcsInfo.URL,
			Digest: map[string]string{"git_commit": vcsInfo.Commit},
		})
		// Remember the URI of the recipe material. We need to find it later to set the index correctly (after sorting)
		recipeMaterialUri = vcsInfo.URL
	}

	// Add all found step containers as materials
	for _, trs := range tr.Status.Steps {
		// image id formats: name@alg:digest
		//                   schema://name@alg:hash
		d := strings.Split(trs.ImageID, "//")
		if len(d) == 2 {
			// Get away with schema
			d[0] = d[1]
		}
		d = strings.Split(d[0], "@")
		if len(d) != 2 {
			continue
		}
		// d[0]: image name
		// d[1]: alg:hash
		h := strings.Split(d[1], ":")
		if len(h) != 2 {
			continue
		}
		att.Predicate.Materials = append(att.Predicate.Materials, in_toto.ProvenanceMaterial{
			URI:    d[0],
			Digest: in_toto.DigestSet{h[0]: h[1]},
		})
	}

	sort.Slice(att.Predicate.Materials, func(i, j int) bool {
		return att.Predicate.Materials[i].URI <= att.Predicate.Materials[j].URI
	})

	if recipeMaterialUri != "" {
		for i, m := range att.Predicate.Materials {
			if m.URI == recipeMaterialUri {
				att.Predicate.Recipe.DefinedInMaterial = intP(i)
				break
			}
		}
	}

	return att, nil
}

func (i *InTotoIte6) Type() PayloadType {
	return PayloadTypeInTotoIte6
}

// getResultDigests scans over the task.Status.TaskSpec.Results to find any
// result that has a name that ends with -DIGEST.
func getResultDigests(tr *v1beta1.TaskRun) []artifactResult {
	results := []artifactResult{}
	// Scan for digests
	for _, r := range tr.Status.TaskSpec.Results {
		if strings.HasSuffix(r.Name, "-DIGEST") {
			// 7 chars in -DIGEST
			results = append(results, artifactResult{
				ParamName:  r.Name[:len(r.Name)-7],
				ResultName: r.Name,
			})
		}
	}

	return results
}

// getVcsInfo scans over the input parameters and looks for parameters
// with specified names.
func getVcsInfo(tr *v1beta1.TaskRun) *vcsInfo {
	var v vcsInfo
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == commitParam {
			v.Commit = p.Value.StringVal
			continue
		}
		if p.Name == urlParam {
			v.URL = p.Value.StringVal
			// make sure url is PURL (git+https)
			if !strings.HasPrefix(v.URL, "git+") {
				v.URL = "git+" + v.URL
			}
			continue
		}
	}

	if v.Commit == "" || v.URL == "" {
		return nil
	}

	return &v
}

// getSubjectDigests uses depends on taskResults with names ending with
// -DIGEST.
// To be able to find the resource that matches the digest, it relies on a
// naming schema for an input parameter.
// If the output result from this task is foo-DIGEST, it expects to find an
// parameter named 'foo', and resolves this value to use for the subject with
// for the found digest.
// If no parameter is found, we search the results of this task, which allows
// for a task to create a random resource not known prior to execution that
// gets checksummed and added as the subject.
// Digests can be on two formats: $alg:$digest (commonly used for container
// image hashes), or $alg:$digest $path, which is used when a step is
// calculating a hash of a previous step.
func getSubjectDigests(tr *v1beta1.TaskRun, results []artifactResult) []in_toto.Subject {
	subs := []in_toto.Subject{}
	r := regexp.MustCompile(`(\S+)\s+(\S+)`)

	// Resolve *-DIGEST variables
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
				// hashalg:hash
				// hashalg:hash filename
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
					// Verify filename is equal to sbuject
					// if groups[2] != sub {
					// 	continue
					// }
					hash = groups[1]
				} else {
					hash = ah[1]
				}

				subs = append(subs, in_toto.Subject{
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
