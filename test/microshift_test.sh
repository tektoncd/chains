#!/usr/bin/env bash
set -e

export namespace="${NAMESPACE:-tekton-chains}"

function install_chains() {
  echo ">> Deploying Tekton Chains"
  ko resolve -f config > release.yaml || fail_test "Tekton Chains build failed"
  yq 'del(.spec.template.spec.containers[]?.securityContext.runAsUser, .spec.template.spec.containers[]?.securityContext.runAsGroup)' release.yaml | kubectl apply -f -

  # Wait for pods to be running in the namespaces we are deploying to
  wait_until_pods_running ${namespace} || fail_test "Tekton Chains did not come up"
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
	oc adm policy add-scc-to-user anyuid -z spire-agent -n spire
	oc adm policy add-scc-to-user anyuid -z spire-server -n spire
	kubectl -n spire apply -f "test/testdata/spire.yaml"
	wait_until_pods_running spire || fail_test "Spire did not come up"
	spire_apply \
		-spiffeID spiffe://example.org/ns/spire/node/example \
		-selector k8s_psat:cluster:example \
		-selector k8s_psat:agent_ns:spire \
		-selector k8s_psat:agent_sa:spire-agent \
		-node
	spire_apply \
		-spiffeID spiffe://example.org/ns/${namespace}/sa/tekton-chains-controller \
		-parentID spiffe://example.org/ns/spire/node/example \
		-selector k8s:ns:${namespace} \
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
	oc adm policy add-scc-to-user anyuid -z vault -n vault
	kubectl -n vault apply -f test/testdata/vault.yaml
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
			bound_subject=spiffe://example.org/ns/${namespace}/sa/tekton-chains-controller \
			token_ttl=15m \
			token_policies=spire-transit
	vault_exec read transit/keys/e2e >/dev/null 2>&1 \
		|| vault_exec write transit/keys/e2e type=ecdsa-p521
	vault_exec read -format=json transit/keys/e2e \
		| jq -r .data.keys.\"1\".public_key >"test/testdata/vault.pub"
}

function chains_patch_spire() {
	kubectl patch -n ${namespace} deployment tekton-chains-controller \
		--patch-file "test/testdata/chains-patch-spire.json"
	# Wait for pods to be running in the namespaces we are deploying to
	wait_until_pods_running ${namespace} || fail_test "Tekton Chains did not come up after patching"
}

function wait_until_pods_running() {
  echo -n "Waiting until all pods in namespace $1 are up"
  for i in {1..150}; do  # timeout after 5 minutes
    local pods="$(kubectl get pods --no-headers -n $1 2>/dev/null)"
    # All pods must be running
    local not_running=$(echo "${pods}" | grep -v Running | grep -v Completed | wc -l)
    if [[ -n "${pods}" && ${not_running} -eq 0 ]]; then
      local all_ready=1
      while read pod ; do
        local status=(`echo -n ${pod} | cut -f2 -d' ' | tr '/' ' '`)
        # All containers must be ready
        [[ -z ${status[0]} ]] && all_ready=0 && break
        [[ -z ${status[1]} ]] && all_ready=0 && break
        [[ ${status[0]} -lt 1 ]] && all_ready=0 && break
        [[ ${status[1]} -lt 1 ]] && all_ready=0 && break
        [[ ${status[0]} -ne ${status[1]} ]] && all_ready=0 && break
      done <<< $(echo "${pods}" | grep -v Completed)
      if (( all_ready )); then
        echo -e "\nAll pods are up:\n${pods}"
        return 0
      fi
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for pods to come up\n${pods}"
  return 1
}

curl https://infra.tekton.dev/tekton-releases/pipeline/latest/release.notags.yaml | yq 'del(.spec.template.spec.containers[]?.securityContext.runAsUser, .spec.template.spec.containers[]?.securityContext.runAsGroup)' | kubectl apply -f -

install_chains

install_spire

install_vault

chains_patch_spire

export GCE_METADATA_HOST=localhost
export OPENSHIFT=localhost
go test -v -count=1 -tags=e2e -timeout=35m ./test/... --kubeconfig ${KUBECONFIG}
