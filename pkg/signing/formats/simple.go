/*
Copyright 2020 The Tekton Authors
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

package formats

import (
	"fmt"
	"path"

	"go.uber.org/zap"

	"github.com/google/go-containerregistry/pkg/name"
)

// SimpleSigning is a formatter that uses the RedHat simple signing format
// https://www.redhat.com/en/blog/container-image-signing
type SimpleSigning struct {
}

// CreatePayload implements the Payloader interface.
func (i *SimpleSigning) CreatePayload(l *zap.SugaredLogger, obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case name.Digest:
		format := newSimpleStruct()
		format.Critical.Identity["docker-reference"] = path.Join(v.RegistryStr(), v.RepositoryStr())
		format.Critical.Image["Docker-manifest-digest"] = v.DigestStr()
		return format, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", v)
	}

}

func newSimpleStruct() simple {
	s := simple{
		Critical: critical{
			Identity: map[string]string{},
			Image:    map[string]string{},
			Type:     "Tekton container signature",
		},
		Optional: map[string]interface{}{},
	}
	return s
}

type simple struct {
	Critical critical
	Optional map[string]interface{}
}

type critical struct {
	Identity map[string]string
	Image    map[string]string
	Type     string
}

func (i *SimpleSigning) Type() PayloadType {
	return PayloadTypeSimpleSigning
}
