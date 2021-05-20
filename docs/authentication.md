# Authentication for Chains

Authentication can be required for the following features in Chains:
* Pushing signatures to an OCI registry after signing an image

This doc will cover how to set this up!

## Authenticating to an OCI Registry
We need to give the `tekton-chains-controller` service account credentials so that the chains controller can push signatures to an OCI registry.

To do this, you will need access to credentials for your registry (they are in a file called `credentials.json` in this example).
Next, create a [Docker config type Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets), which will contain the credentials required to push signatures:

```
kubectl create secret docker-registry registry-credentials \
  --docker-server=gcr.io \
  --docker-username=_json_key \
  --docker-email=someemail@something.com \
  --docker-password="$(cat credentials.json)" \
  -n tekton-chains
```
More details around creating this secret can be found [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).

We can then give the `tekton-chains-controller` service account access to these credentials:

```
kubectl patch serviceaccount tekton-chains-controller \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}" -n tekton-chains
```

Now, `tekton-chains-controller` should be able to push to your OCI registry.
