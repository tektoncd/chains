# Hashicorp Integration with Tekton Chains

### In this tutorial, we will be running [Chains Signed Provenance Tutorial](https://github.com/tektoncd/chains/blob/main/docs/tutorials/signed-provenance-tutorial.md) using `kms` solution by integrating Tekton Chains with Hashicorp Vault

**NOTE**
* Prerequisite
    * Hashicorp vault should be installed
        * If not installed, you can also try on a minikube or a kind cluster. For more info see [here](https://developer.hashicorp.com/vault/tutorials/kubernetes/kubernetes-minikube-raft)  
        * This provider also requires that the `transit` secret engine is enabled
        * If not done, you can login into the vault provider and run the following command

          ```bash=
          $ vault secrets enable transit
          Success! Enabled the transit secrets engine at: transit/
          ```
    * You can get more info about `transit` secrets [here](https://developer.hashicorp.com/vault/docs/secrets/transit#setup)

    * Make sure Tekton Pipelines and Tekton Chains is installed


### [Chains Signed Provenance Tutorial](https://github.com/tektoncd/chains/blob/main/docs/tutorials/signed-provenance-tutorial.md) using `kms` solution by integrating with Hashicorp Vault

### Step 1: Generate a Key Pair

This provider requires that the standard Vault environment variables (`$VAULT_ADDR, $VAULT_TOKEN`) are set correctly.

```bash=
$ export VAULT_ADDR=http://localhost:8200
$ export VAULT_TOKEN=testtoken
$ vault secrets enable transit  ---> Ignore if you have already done
```

Now run the following command

```bash=
$ cosign generate-key-pair --kms hashivault://$keyname
```

**NOTE**:- If you enabled transit secret engine at different path with the use of -path flag (i.e., $ vault secrets enable -path="someotherpath" transit), you can use TRANSIT_SECRET_ENGINE_PATH environment variable to specify this path while generating a key pair like the following:

In that case the command will be

```bash=
$ TRANSIT_SECRET_ENGINE_PATH="someotherpath" cosign generate-key-pair --kms hashivault://$keyname
```

### Step 2: Set up Authenticaion

There are two forms of authentication that need to be set up:

1.  The Chains controller will be pushing signatures to an OCI registry using the credentials linked to your TaskRun's service account. See our authentication [doc](https://github.com/tektoncd/chains/blob/main/docs/authentication.md)
2.  The Kaniko Task that will build and push the image needs push permissions for your registry.

To set up auth for the Kaniko Task, you'll need a Kubernetes secret of a docker config.json file which contains the required auth. You can create the secret by running:

```bash=
kubectl create secret generic [DOCKERCONFIG_SECRET_NAME] --from-file [PATH TO CONFIG.JSON]
```

### Step 3: Configuring Tekton Chains

You'll need to make these changes to the Tekton Chains configMap i.e. `chains-config` configMap:

```yaml
* artifacts.taskrun.format: slsa/v1
* artifacts.taskrun.storage: oci
* artifacts.taskrun.signer: kms
* artifacts.pipelinerun.signer: kms
* artifacts.oci.signer: kms
* transparency.enabled: "true"
* signers.kms.kmsref: hashivault://$keyname
* signers.kms.auth.address: <VAULT_ADDR>
* signers.kms.auth.token: <VAULT_TOKEN>
```

### Step 4: Start the Kaniko Task

* First apply the

```bash
kubectl apply -f examples/kaniko/kaniko.yaml
```
Substitute with the URI or file path to your Kaniko task.

* Set the following enviornment variables:

```bash=
export REGISTRY=<url_of_registry>
export DOCKERCONFIG_SECRET_NAME=<name_of_the_secret_in_docker_config_json>
```

Substitute with the URL of the registry where you want to push the image.
Substitute with the name of the secret in the docker config.json file.

* Start the Kaniko Task

```bash=
tkn task start --param IMAGE=$REGISTRY/kaniko-chains --use-param-defaults --workspace name=source,emptyDir="" --workspace name=dockerconfig,secret=$DOCKERCONFIG_SECRET_NAME kaniko-chains
```

* Wait for a minute to allow Tekton Chains to generate the provenance and sign it, and then check the availability of the chains.tekton.dev/signed=true annotation on the task run.

```bash=
kubectl get tr <task_run_name> -o json | jq -r .metadata.annotations
{
	"chains.tekton.dev/signed": "true",
	...
}
```

### Step 5: Verify the image and the attestation

```bash=
cosign verify --key cosign.pub $REGISTRY/kaniko-chains
cosign verify-attestation --key cosign.pub --type slsaprovenance $REGISTRY/kaniko-chains
```

or you can use the hashivault://$keyname as key as well

```bash=
cosign verify --key hashivault://testkey $REGISTRY/kaniko-chains
cosign verify-attestation --key hashivault://testkey --type slsaprovenance $REGISTRY/kaniko-chains
```


The output would be like this

```bash=
Verification for index.docker.io/$REGISTRY/kaniko-chains:latest --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - The claims were present in the transparency log
  - The signatures were integrated into the transparency log when the certificate was valid
  - The signatures were verified against the specified public key

[{"critical":{"identity":{"docker-reference":"index.docker.io/$REGISTRY/kaniko-chains"},"image":{"docker-manifest-digest":"sha256:e14396b283abcbacddba403a923a7fdecf2c54537a1d6a1ee1076767bec742d1"},"type":"cosign container image signature"},"optional":null}]

Verification for docker.io/$REGISTRY/kaniko-chains --
The following checks were performed on each of these signatures:
  - The cosign claims were validated
  - The claims were present in the transparency log
  - The signatures were integrated into the transparency log when the certificate was valid
  - The signatures were verified against the specified public key
{"payloadType":"application/vnd.in-toto+json","payload":"eyJfdHlwZSI6Imh0dHBzOi8vaW4tdG90by5pby9TdGF0ZW1lbnQvdjAuMSIsInByZWRpY2F0ZVR5cGUiOiJodHRwczovL3Nsc2EuZGV2L3Byb3ZlbmFuY2UvdjAuMiIsInN1YmplY3QiOlt7Im5hbWUiOiJpbmRleC5kb2NrZXIuaW8vcHVuZWV0MjE0Ny9rYW5pa28tY2hhaW5zIiwiZGlnZXN0Ijp7InNoYTI1NiI6ImUxNDM5NmIyODNhYmNiYWNkZGJhNDAzYTkyM2E3ZmRlY2YyYzU0NTM3YTFkNmExZWUxMDc2NzY3YmVjNzQyZDEifX1dLCJwcmVkaWNhdGUiOnsiYnVpbGRlciI6eyJpZCI6Imh0dHBzOi8vdGVrdG9uLmRldi9jaGFpbnMvdjIifSwiYnVpbGRUeXBlIjoidGVrdG9uLmRldi92MWJldGExL1Rhc2tSdW4iLCJpbnZvY2F0aW9uIjp7ImNvbmZpZ1NvdXJjZSI6e30sInBhcmFtZXRlcnMiOnsiQlVJTERFUl9JTUFHRSI6Imdjci5pby9rYW5pa28tcHJvamVjdC9leGVjdXRvcjp2MS41LjFAc2hhMjU2OmM2MTY2NzE3ZjdmZTBiN2RhNDQ5MDhjOTg2MTM3ZWNmZWFiMjFmMzFlYzM5OTJmNmUxMjhmZmY4YTk0YmU4YTUiLCJDT05URVhUIjoiLi8iLCJET0NLRVJGSUxFIjoiLi9Eb2NrZXJmaWxlIiwiRVhUUkFfQVJHUyI6IiIsIklNQUdFIjoiZG9ja2VyLmlvL3B1bmVldDIxNDcva2FuaWtvLWNoYWlucyJ9LCJlbnZpcm9ubWVudCI6eyJhbm5vdGF0aW9ucyI6eyJwaXBlbGluZS50ZWt0b24uZGV2L3JlbGVhc2UiOiJjODAyMDY5In0sImxhYmVscyI6eyJhcHAua3ViZXJuZXRlcy5pby9tYW5hZ2VkLWJ5IjoidGVrdG9uLXBpcGVsaW5lcyIsInRla3Rvbi5kZXYvdGFzayI6Imthbmlrby1jaGFpbnMifX19LCJidWlsZENvbmZpZyI6eyJzdGVwcyI6W3siZW50cnlQb2ludCI6InNldCAtZVxuZWNobyBcIkZST00gYWxwaW5lQHNoYTI1Njo2OWU3MGE3OWYyZDQxYWI1ZDYzN2RlOThjMWUwYjA1NTIwNmJhNDBhODE0NWU3YmRkYjU1Y2NjMDRlMTNjZjhmXCIgfCB0ZWUgLi9Eb2NrZXJmaWxlXG4iLCJhcmd1bWVudHMiOm51bGwsImVudmlyb25tZW50Ijp7ImNvbnRhaW5lciI6ImFkZC1kb2NrZXJmaWxlIiwiaW1hZ2UiOiJkb2NrZXIuaW8vbGlicmFyeS9iYXNoQHNoYTI1NjoxZWEzMGQ5YjY1Nzk3ZmJhZTQ3ODdmNjE4ODc5NmU3MTg5MzcxMDE5MDMxOTU4YTE2NzQyM2QzNDdkMzJlYWRhIn0sImFubm90YXRpb25zIjpudWxsfSx7ImVudHJ5UG9pbnQiOiIiLCJhcmd1bWVudHMiOlsiIiwiLS1kb2NrZXJmaWxlPS4vRG9ja2VyZmlsZSIsIi0tY29udGV4dD0vd29ya3NwYWNlL3NvdXJjZS8uLyIsIi0tZGVzdGluYXRpb249ZG9ja2VyLmlvL3B1bmVldDIxNDcva2FuaWtvLWNoYWlucyIsIi0tZGlnZXN0LWZpbGU9L3Rla3Rvbi9yZXN1bHRzL0lNQUdFX0RJR0VTVCJdLCJlbnZpcm9ubWVudCI6eyJjb250YWluZXIiOiJidWlsZC1hbmQtcHVzaCIsImltYWdlIjoiZ2NyLmlvL2thbmlrby1wcm9qZWN0L2V4ZWN1dG9yQHNoYTI1NjpjNjE2NjcxN2Y3ZmUwYjdkYTQ0OTA4Yzk4NjEzN2VjZmVhYjIxZjMxZWMzOTkyZjZlMTI4ZmZmOGE5NGJlOGE1In0sImFubm90YXRpb25zIjpudWxsfSx7ImVudHJ5UG9pbnQiOiJzZXQgLWVcbmVjaG8gZG9ja2VyLmlvL3B1bmVldDIxNDcva2FuaWtvLWNoYWlucyB8IHRlZSAvdGVrdG9uL3Jlc3VsdHMvSU1BR0VfVVJMXG4iLCJhcmd1bWVudHMiOm51bGwsImVudmlyb25tZW50Ijp7ImNvbnRhaW5lciI6IndyaXRlLXVybCIsImltYWdlIjoiZG9ja2VyLmlvL2xpYnJhcnkvYmFzaEBzaGEyNTY6MWVhMzBkOWI2NTc5N2ZiYWU0Nzg3ZjYxODg3OTZlNzE4OTM3MTAxOTAzMTk1OGExNjc0MjNkMzQ3ZDMyZWFkYSJ9LCJhbm5vdGF0aW9ucyI6bnVsbH1dfSwibWV0YWRhdGEiOnsiYnVpbGRTdGFydGVkT24iOiIyMDIzLTA3LTE4VDA4OjM4OjU0WiIsImJ1aWxkRmluaXNoZWRPbiI6IjIwMjMtMDctMThUMDg6Mzk6MTNaIiwiY29tcGxldGVuZXNzIjp7InBhcmFtZXRlcnMiOmZhbHNlLCJlbnZpcm9ubWVudCI6ZmFsc2UsIm1hdGVyaWFscyI6ZmFsc2V9LCJyZXByb2R1Y2libGUiOmZhbHNlfSwibWF0ZXJpYWxzIjpbeyJ1cmkiOiJkb2NrZXIuaW8vbGlicmFyeS9iYXNoIiwiZGlnZXN0Ijp7InNoYTI1NiI6IjFlYTMwZDliNjU3OTdmYmFlNDc4N2Y2MTg4Nzk2ZTcxODkzNzEwMTkwMzE5NThhMTY3NDIzZDM0N2QzMmVhZGEifX0seyJ1cmkiOiJnY3IuaW8va2FuaWtvLXByb2plY3QvZXhlY3V0b3IiLCJkaWdlc3QiOnsic2hhMjU2IjoiYzYxNjY3MTdmN2ZlMGI3ZGE0NDkwOGM5ODYxMzdlY2ZlYWIyMWYzMWVjMzk5MmY2ZTEyOGZmZjhhOTRiZThhNSJ9fV19fQ==","signatures":[{"keyid":"SHA256:wvNLyVMa1zxAWD9ZjvKanoCuukphRbKYdLM24TEEAj0","sig":"MEYCIQDsMLBOWKZKDBiiVJOz4ZQbPTKfQwhdBgsbVupJlvlN+gIhAPxMbCfjKGSl1ity9RS9/UMXRcI5QtkCH+LX6t4V5/Ft"}]}

```