package patch

import "encoding/json"

// GetAnnotationsPatch returns patch bytes that can be used with kubectl patch
func GetAnnotationsPatch(newAnnotations map[string]string) ([]byte, error) {
	p := patch{
		Metadata: metadata{
			Annotations: newAnnotations,
		},
	}
	return json.Marshal(p)
}

// These are used to get proper json formatting
type patch struct {
	Metadata metadata `json:"metadata,omitempty"`
}
type metadata struct {
	Annotations map[string]string `json:"annotations,omitempty"`
}
