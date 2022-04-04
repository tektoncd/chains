#!/usr/bin/env bash

# Copyright 2020 The Tekton Authors
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

# This script calls out to scripts in tektoncd/plumbing to setup a cluster
# and deploy Tekton Pipelines to it for running integration tests.

source $(git rev-parse --show-toplevel)/test/e2e-common.sh

# Script entry point.

initialize $@

header "Setting up environment"

# Test against nightly instead of latest.
install_tkn

export RELEASE_YAML="https://storage.googleapis.com/tekton-releases-nightly/pipeline/latest/release.yaml"
install_pipeline_crd

install_chains

install_spire

install_vault

chains_patch_spire

failed=0

# Run the integration tests
header "Running Go e2e tests"
go_test_e2e -timeout=20m ./test/... || failed=1

(( failed )) && dump_logs && fail_test
success
