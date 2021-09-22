# Authentication for Chains

Authentication must be set up to take advantage of the following features in Chains:
* Pushing signatures to an OCI registry after signing an image

This doc will cover how to set this up!

## Authenticating to an OCI Registry
The Chains controller will use the same service account your Task runs under as credentials for pushing signatures to an OCI registry. This section will cover how to set up a service account that has the necessary credentials.

First, you will need access to credentials for your registry (they are in a file called `credentials.json` in this example).
Next, create a [Docker config type Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets), which will contain the credentials required to push signatures:


Set the namespace and name of the Kubernetes service account:

```
export NAMESPACE=<your namespace>
export SERVICE_ACCOUNT_NAME=<service account name>
```

Then, create a `.dockerconfig` type secret:

```
kubectl create secret docker-registry registry-credentials \
  --docker-server=gcr.io \
  --docker-username=_json_key \
  --docker-email=someemail@something.com \
  --docker-password="$(cat credentials.json)" \
  -n $NAMESPACE
```
More details around creating this secret can be found [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).


Finally, give the service account access to the secret above:

```
kubectl patch serviceaccount $SERVICE_ACCOUNT_NAME \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}" -n $NAMESPACE
```

Now, Chains has push permissions for any TaskRuns running under the service account `$SERVICE_ACCOUNT_NAME`.
