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

package metadata

import (
	slsa "github.com/in-toto/attestation/go/predicates/provenance/v1"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GetBuildMetadata returns SLSA metadata.
func GetBuildMetadata(obj objects.TektonObject) *slsa.BuildMetadata {
	var startedOn *timestamppb.Timestamp
	var finishedOn *timestamppb.Timestamp
	objStartTime := obj.GetStartTime()
	objCompletitionTime := obj.GetCompletitionTime()

	if objStartTime != nil {
		startedOn = timestamppb.New(*objStartTime)
	}

	if objCompletitionTime != nil {
		finishedOn = timestamppb.New(*objCompletitionTime)
	}

	return &slsa.BuildMetadata{
		InvocationId: string(obj.GetUID()),
		StartedOn:    startedOn,
		FinishedOn:   finishedOn,
	}
}
