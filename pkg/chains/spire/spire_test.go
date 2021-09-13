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

package spire

import (
	"testing"

	logtesting "knative.dev/pkg/logging/testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestVerify(t *testing.T) {
	tr := &v1beta1.TaskRun{
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				TaskRunResults: []v1beta1.TaskRunResult{
					{
						Name:  "SVID",
						Value: svid,
					},
				},
			},
		},
	}
	logger := logtesting.TestLogger(t)
	if err := Verify(tr, logger); err == nil {
		t.Fatal("no results to verify, this should fail")
	}

	digestTaskRunResults := []v1beta1.TaskRunResult{
		{
			Name:  "IMAGE_DIGEST",
			Value: "sha256:52a9b14f0938cd8b0b1bc29776fb52b6f108ee47363a572d5d07fbbd3f12e0b3",
		}, {
			Name:  "IMAGE_DIGEST.sig",
			Value: "MEUCIBcrPe52qLou+vDlaJ3sOCi1e1soT7IYbK0Guc8NVcWOAiEAtJDi6mqAY1O3L/AP0fNhWUvqii4bXs+XezGwtHo5VPQ=",
		},
	}
	tr.Status.TaskRunResults = append(tr.Status.TaskRunResults, digestTaskRunResults...)
	if err := Verify(tr, logger); err != nil {
		t.Fatal(err)
	}
}

const svid = `-----BEGIN CERTIFICATE-----
MIICtzCCAZ+gAwIBAgIQGaiyrxX0Zvf7Bv+CU123qDANBgkqhkiG9w0BAQsFADAe
MQswCQYDVQQGEwJVUzEPMA0GA1UEChMGU1BJRkZFMB4XDTIxMDYxNjIwMjcxOVoX
DTIxMDYxNjIxMjcyOVowHTELMAkGA1UEBhMCVVMxDjAMBgNVBAoTBVNQSVJFMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAENygiz0a2vL2Q0Pg/Sl7I9ab/VX+ZGzjU
yDlmEA2ANpYlpaR4u8NuhiI7SSajC95aeiRAG7Wv1EHfjcI4fJthuqOBvDCBuTAO
BgNVHQ8BAf8EBAMCA6gwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMAwG
A1UdEwEB/wQCMAAwHQYDVR0OBBYEFIM3W+xFL1Dn0BQ890zLbq3c/twsMB8GA1Ud
IwQYMBaAFOgVB4NKkNjLX0KK2nwCAcPYYMDyMDoGA1UdEQQzMDGGL3NwaWZmZTov
L29pZGMuZGxvcmVuYy5kZXYvbnMvZGVmYXVsdC9zYS9kZWZhdWx0MA0GCSqGSIb3
DQEBCwUAA4IBAQAp6nRqhULi4mjZ7JIFxrabo0xPCGwCoOXW8IVHYCkyROZXZWrq
JLHKkxur0iJqjOjhUqXhtGZ/TWdhL0oNEGCBs543dOnBYYYje+41jyPLR4QbKyJg
juFnBH3xOD0w9hHIkDzTBn7aQmIkETmMzeYFtfFUTAClgi7ATZwm9kGGK82L2oBi
YuHXBuxBkadO8yKN21tPrNLVMvhGEg9igJgY3Rs6coVwlNBCwINUe6Eql20+exzO
nz4AZIihdYP9a2W7qpPdiUk7pUvA6i9nepMVBCIY5LXRU5YhD4TSJxOPacwQP6IB
bQMsVW0vU7HuMxsI/kZZwN4I0knvGThCrMdp
-----END CERTIFICATE-----`
