#!/usr/bin/env bash

# Copyright 2019 The Tekton Authors
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

# This script runs the presubmit tests; it is started by prow for each PR.
# For convenience, it can also be executed manually.
# Running the script without parameters, or with the --all-tests
# flag, causes all tests to be executed, in the right order.
# Use the flags --build-tests, --unit-tests and --integration-tests
# to run a specific set of tests.

# Helper functions for E2E tests.

# Check if we have a specific RELEASE_YAML global environment variable to use
# instead of detecting the latest released one from tektoncd/pipeline releases
RELEASE_YAML=${RELEASE_YAML:-}

source $(dirname $0)/../vendor/github.com/tektoncd/plumbing/scripts/e2e-tests.sh

function install_tkn() {
  echo ">> Installing tkn"
  TKN_VERSION=0.20.0
  # Get the tar.xz
  curl -LO https://github.com/tektoncd/cli/releases/download/v$TKN_VERSION/tkn_$TKN_VERSION_Linux_x86_64.tar.gz
  # Extract tkn to your PATH (e.g. /usr/local/bin)
  tar xvzf tkn_$TKN_VERSION_Linux_x86_64.tar.gz -C /usr/local/bin/ tkn
}

function install_pipeline_crd() {
  local latestreleaseyaml
  echo ">> Deploying Tekton Pipelines"
  if [[ -n ${RELEASE_YAML} ]];then
	  latestreleaseyaml=${RELEASE_YAML}
  else
    latestreleaseyaml=$(curl -s https://api.github.com/repos/tektoncd/pipeline/releases|python -c "import sys, json;x=json.load(sys.stdin);ass=x[0]['assets'];print([ x['browser_download_url'] for x in ass if x['name'] == 'release.yaml'][0])")
  fi
  [[ -z ${latestreleaseyaml} ]] && fail_test "Could not get latest released release.yaml"
  kubectl apply -f ${latestreleaseyaml} ||
    fail_test "Tekton pipeline installation failed"

  # Make sure that eveything is cleaned up in the current namespace.
  for res in pipelineresources tasks pipelines taskruns pipelineruns; do
    kubectl delete --ignore-not-found=true ${res}.tekton.dev --all
  done

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-pipelines || fail_test "Tekton Pipeline did not come up"
}

function install_chains() {
  echo ">> Deploying Tekton Chains"
  ko apply -f config/ || fail_test "Tekton Chains installation failed"

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-chains || fail_test "Tekton Chains did not come up"
}

function chains_patch_spire() {
  kubectl patch -n tekton-chains deployment tekton-chains-controller \
    --patch-file "$(dirname $0)/testdata/chains-patch-spire.json"
  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running tekton-chains || fail_test "Tekton Chains did not come up after patching"
}

function dump_logs() {
  echo ">> Tekton Chains Logs"
  kubectl logs deployment/tekton-chains-controller -n tekton-chains
}

function spire_apply() {
  if [ $# -lt 2 -o "$1" != "-spiffeID" ]; then
    echo "spire_apply requires a spiffeID as the first arg" >&2
    exit 1
  fi
  show=$(kubectl exec -n spire spire-server-0 -c spire-server -- \
    /opt/spire/bin/spire-server entry show $1 $2)
  if [ "$show" != "Found 0 entries" ]; then
    # delete to recreate
    entryid=$(echo "$show" | grep "^Entry ID" | cut -f2 -d:)
    kubectl exec -n spire spire-server-0 -c spire-server -- \
      /opt/spire/bin/spire-server entry delete -entryID $entryid
  fi
  kubectl exec -n spire spire-server-0 -c spire-server -- \
    /opt/spire/bin/spire-server entry create "$@"
}

function install_spire() {
  echo ">> Deploying Spire"
  kubectl create ns spire --dry-run=client -o yaml | kubectl apply -f -
  kubectl -n spire apply -f "$(dirname $0)/testdata/spire.yaml"
  wait_until_pods_running spire || fail_test "Spire did not come up"
  spire_apply \
    -spiffeID spiffe://example.org/ns/spire/node/example \
    -selector k8s_psat:cluster:example \
    -selector k8s_psat:agent_ns:spire \
    -selector k8s_psat:agent_sa:spire-agent \
    -node
  spire_apply \
    -spiffeID spiffe://example.org/ns/tekton-chains/sa/tekton-chains-controller \
    -parentID spiffe://example.org/ns/spire/node/example \
    -selector k8s:ns:tekton-chains \
    -selector k8s:sa:tekton-chains-controller  
}

function vault_exec() {
  envcmd=""
  if [ -n "$ROOT_TOKEN" ]; then
    envcmd="env VAULT_TOKEN=$ROOT_TOKEN"
  fi
  kubectl exec -i -n vault vault-0 -- $envcmd vault "$@"
}

function install_vault() {
  echo ">> Deploying Vault"
  kubectl create ns vault --dry-run=client -o yaml | kubectl apply -f -
  kubectl -n vault apply -f "$(dirname $0)/testdata/vault.yaml"
  wait_until_pods_running vault || fail_test "Vault did not come up"
  ROOT_TOKEN=token12345
  vault_exec secrets list 2>&1 | grep "^transit/" \
    || vault_exec secrets enable transit
  vault_exec auth list 2>&1 | grep "^jwt/" \
    || vault_exec auth enable jwt
  vault_exec read auth/jwt/config >/dev/null 2>&1 \
    || vault_exec write auth/jwt/config \
      oidc_discovery_url=http://spire-oidc.spire:8082 \
      default_role="spire"
  vault_exec policy read spire-transit >/dev/null 2>&1 \
    || vault_exec policy write spire-transit - <<EOF
path "transit/*" {
  capabilities = ["read"]
}
path "transit/sign/e2e" {
  capabilities = ["create", "read", "update"]
}
path "transit/sign/e2e/*" {
  capabilities = ["read", "update"]
}
path "transit/verify/e2e" {
  capabilities = ["create", "read", "update"]
}
path "transit/verify/e2e/*" {
  capabilities = ["read", "update"]
}
EOF
  vault_exec read auth/jwt/role/spire-chains-controller >/dev/null 2>&1 \
    || vault_exec write auth/jwt/role/spire-chains-controller \
      role_type=jwt \
      user_claim=sub \
      bound_audiences=e2e \
      bound_subject=spiffe://example.org/ns/tekton-chains/sa/tekton-chains-controller \
      token_ttl=15m \
      token_policies=spire-transit
  vault_exec read transit/keys/e2e >/dev/null 2>&1 \
    || vault_exec write transit/keys/e2e type=ecdsa-p521
  vault_exec read -format=json transit/keys/e2e \
    | jq -r .data.keys.\"1\".public_key >"$(dirname $0)/testdata/vault.pub"
}