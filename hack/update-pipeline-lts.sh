#!/usr/bin/env bash

# Copyright 2026 The Tekton Authors
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

# Fetch the current list of supported Pipeline LTS versions from
# tektoncd/pipeline's releases.md and write them to .github/pipeline-lts.json.
#
# The releases.md file is the single source of truth for which Pipeline
# versions are actively supported. This script extracts the latest patch
# tag for each active LTS release.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OUTPUT="${REPO_ROOT}/.github/pipeline-lts.json"

RELEASES_URL="https://raw.githubusercontent.com/tektoncd/pipeline/main/releases.md"

echo "Fetching releases.md from tektoncd/pipeline..."
RELEASES_MD=$(curl -sSfL "${RELEASES_URL}")

echo "Extracting active LTS versions..."
# Parse releases.md: find LTS entries under the "## Release" section (not "## End of Life"),
# then grab the "Latest Release" tag from the next line.
VERSIONS=$(echo "${RELEASES_MD}" | awk '
  /^## Release$/ { found=1; next }
  found && /^## / { exit }
  found && /^### v[0-9]+\.[0-9]+ \(LTS\)/ { lts=1; next }
  lts && /Latest Release/ { match($0, /v[0-9]+\.[0-9]+\.[0-9]+/); print substr($0, RSTART, RLENGTH); lts=0 }
')

if [[ -z "${VERSIONS}" ]]; then
  echo "error: failed to extract any LTS versions from releases.md"
  exit 1
fi

# Build JSON array
JSON=$(echo "${VERSIONS}" | jq -R -s 'split("\n") | map(select(length > 0))')

echo "Writing ${OUTPUT}..."
echo "${JSON}" | jq . > "${OUTPUT}"

echo "Active Pipeline LTS versions:"
echo "${JSON}" | jq -r '.[]'
