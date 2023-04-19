# Copyright 2023 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

export IMAGE_LOC=$(tkn pr describe --last -o jsonpath="{.status.results[1].value.uri}")
export IMAGE_SHA=$(tkn pr describe --last -o jsonpath="{.status.results[1].value.digest}")
export IMAGE_REF=$IMAGE_LOC@$IMAGE_SHA
gcloud artifacts docker images describe $IMAGE_REF --show-all-metadata --format json | jq -r '.provenance_summary.provenance[0].envelope.payload' | tr '\-_' '+/' | base64 -d | jq > provenance.json
gcloud artifacts docker images describe $IMAGE_REF --show-all-metadata --format json | jq -r '.provenance_summary.provenance[0].envelope.signatures[0].sig' | tr '\-_' '+/' | base64 -d > signature
