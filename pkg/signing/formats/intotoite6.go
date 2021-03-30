package formats

import (
	"fmt"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type InTotoIte6 struct {
}

type artifactResult struct {
	ParamName  string
	ResultName string
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
	// Params with name CHAINS-GIT_* -> Materials
	att := in_toto.Provenance{
		Attestation: in_toto.Attestation{
			AttestationType: in_toto.ProvenanceTypeV1,
		},
		Builder: in_toto.Builder{
			ID: tr.Status.PodName,
		},
	}

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

	att.Subject = in_toto.ArtifactCollection{}
	att.Materials = in_toto.ArtifactCollection{}
	// Resolve *-DIGEST variables
	for _, ar := range results {
		// get subject
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
			// Parameter was not specifed for this task run
			continue
		}
		for _, trr := range tr.Status.TaskRunResults {
			if trr.Name == ar.ResultName {
				// Value should be on format:
				// hashalg:hash
				d := strings.Split(trr.Value, ":")
				if len(d) != 2 {
					continue
				}

				att.Subject[sub] = in_toto.ArtifactDigest{
					d[0]: d[1],
				}
			}
		}
	}

	git_hash := ""
	git_url := ""
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == "CHAINS-GIT_COMMIT" {
			git_hash = p.Value.StringVal
			continue
		}
		if p.Name == "CHAINS-GIT_URL" {
			git_url = p.Value.StringVal
			// make sure url is PURL (git+https)
			if !strings.HasPrefix(git_url, "git+") {
				git_url = "git+" + git_url
			}
			continue
		}
	}
	if git_hash != "" && git_url != "" {
		att.Materials[git_url] = in_toto.ArtifactDigest{
			"git_commit": git_hash,
		}
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
		att.Materials[d[0]] = in_toto.ArtifactDigest{
			h[0]: h[1],
		}
	}

	return att, nil
}

func (i *InTotoIte6) Type() PayloadType {
	return PayloadTypeInTotoIte6
}
