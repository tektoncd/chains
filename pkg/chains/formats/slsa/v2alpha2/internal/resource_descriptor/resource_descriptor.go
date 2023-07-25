/*
Copyright 2023 The Tekton Authors
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

package resourcedescriptor

// types for fields in ResourceDescriptor
// https://github.com/in-toto/attestation/blob/main/spec/v1/resource_descriptor.md
type Name string
type MediaType string

const (
	// PipelineConfigName is the name of the resolved dependency of the pipelineRef.
	PipelineConfigName Name = "pipeline"
	// TaskConfigName is the name of the resolved dependency of the top level taskRef.
	TaskConfigName Name = "task"
	// PipelineTaskConfigName is the name of the resolved dependency of the pipeline task.
	PipelineTaskConfigName Name = "pipelineTask"
	// InputResultName is the name of the resolved dependency generated from Type hinted parameters or results.
	InputResultName Name = "inputs/result"
	// PipelineResourceName is the name of the resolved dependency of pipeline resource.
	PipelineResourceName Name = "pipelineResource"
	// PipelineRunResults is the string used to format the name of a pipelinerun result.
	PipelineRunResults Name = "pipelineRunResults/%s"
	// TaskRunResults is the string used to format the name of a taskrun result.
	TaskRunResults Name = "taskRunResults/%s"
	// JsonMediaType is the media type of json encoded content used in resource descriptors
	JsonMediaType MediaType = "application/json"
)
