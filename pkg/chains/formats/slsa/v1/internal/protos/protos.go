package protos

import (
	"encoding/json"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetPredicateStruct returns a protobuf struct from the given SLSAv0.2 predicate.
func GetPredicateStruct(predicate *slsa.ProvenancePredicate) (*structpb.Struct, error) {
	predicateJSON, err := json.Marshal(predicate)
	if err != nil {
		return nil, err
	}

	predicateStruct := &structpb.Struct{}
	err = protojson.Unmarshal(predicateJSON, predicateStruct)
	if err != nil {
		return nil, err
	}

	return predicateStruct, nil
}
