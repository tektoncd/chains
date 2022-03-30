//go:build tools

/*
Copyright 2019 The Tekton Authors

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

package tools

// This is needed to force "go mod" to vendor these modules, which we only use for scripts.
import (
	_ "github.com/tektoncd/plumbing"
	_ "github.com/tektoncd/plumbing/scripts"

	_ "k8s.io/code-generator/cmd/deepcopy-gen"

	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/addlicense"
	_ "github.com/google/go-licenses/licenses"
)
