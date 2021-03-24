package formats

import (
	"fmt"

	"github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type InTotoIte6 struct {
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
	// Input Resource Results -> Materials
	// Output Resource Results -> Subjects
	// The entire TaskRun body -> Recipe.Arguments

	att := in_toto.Provenance{
		Attestation: in_toto.Attestation{
			AttestationType: in_toto.ProvenanceTypeV1,
		},
	}

	// Populate materials with resource inputs.
	att.Subject = in_toto.ArtifactCollection{}
	att.Materials = in_toto.ArtifactCollection{}
	if tr.Spec.Resources != nil {
		for _, r := range tr.Spec.Resources.Inputs {
			fmt.Printf("ITE6: resource input: %s\n", r.Name)
			for _, rr := range tr.Status.ResourcesResult {
				if r.Name == rr.ResourceName {
					// if _, ok := l.Materials[rr.ResourceName]; !ok {
					// 	l.Materials[rr.ResourceName] = map[string]string{}
					// }
					// m := l.Materials[rr.ResourceName].(map[string]string)
					// m[rr.Key] = rr.Value
					fmt.Println("ITE6: match")
				}
			}
		}

		// Dummy to just loop over the status results
		for _, rr := range tr.Status.ResourcesResult {
			fmt.Printf("  ITE6: resource result %s\n", rr.ResourceName)
			fmt.Printf("  ITE6: resource result key %s\n", rr.Key)
			fmt.Printf("  ITE6: resource result value %s\n", rr.Value)
			fmt.Printf("  ITE6: resource result type %s\n", rr.ResultType)
		}

		// Populate products with resource outputs.
		for _, r := range tr.Spec.Resources.Outputs {
			fmt.Printf("ITE6: resource output: %s\n", r.Name)
			for _, rr := range tr.Status.ResourcesResult {
				if r.Name == rr.ResourceName {
					// if _, ok := l.Products[rr.ResourceName]; !ok {
					// 	l.Products[rr.ResourceName] = map[string]string{}
					// }
					// m := l.Products[rr.ResourceName].(map[string]string)
					// m[rr.Key] = rr.Value
					fmt.Println("ITE: 6 match")
				}
			}
		}
	} else {
		fmt.Println("ITE6: No resources found")
	}

	return att, nil
}

func (i *InTotoIte6) Type() PayloadType {
	return PayloadTypeInTotoIte6
}
