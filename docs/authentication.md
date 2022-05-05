<!--
---
linkTitle: "Authentication for Chains"
weight: 10
---
-->

# Authentication for Chains

Authentication must be set up to take advantage of the following features in Chains:

* Pushing signatures to an OCI registry after signing an image
* Using Fulcio to get Signing Certificates when utilizing Keyless signing.

This doc will cover how to set this up!

## Authenticating to an OCI Registry

The Chains controller will use the same service account your Task runs under as credentials for pushing signatures to an OCI registry. This section will cover how to set up a service account that has the necessary credentials.

Set the namespace and name of the Kubernetes service account:

```shell
export NAMESPACE=<your namespace>
export SERVICE_ACCOUNT_NAME=<service account name>
```

### Create a Secret based on existing credentials

If you already ran `docker login`, you can copy the credentials stored in
config.json into Kubernetes.

> Note: Make sure that any external credentials store, such as the native
> keychain of the operating system, is not used to store the credentials and
> the config.json is of the format:
> ```json
> {
>   "auths": {
>     "<registry>": {
>       "auth": "redacted"
>     }
>   }
> }
> ```

Create a secret with config.json:

```shell
kubectl create secret generic docker-registry \
    --from-file=.dockerconfigjson=<path/to/.docker/config.json> \
    --type=kubernetes.io/dockerconfigjson
    -n $NAMESPACE
```

More details around creating this secret can be found [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).

### Create a Secret by providing credentials on the command line

First, you will need access to credentials for your registry (they are in a file called `credentials.json` in this example).
Next, create a [Docker config type Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets), which will contain the credentials required to push signatures:

Then, create a `.dockerconfig` type secret:

```shell
kubectl create secret docker-registry registry-credentials \
  --docker-server=gcr.io \
  --docker-username=_json_key \
  --docker-email=someemail@something.com \
  --docker-password="$(cat credentials.json)" \
  -n $NAMESPACE
```

More details around creating this secret can be found [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#create-a-secret-by-providing-credentials-on-the-command-line).

### Grant access to the service account

Finally, give the service account access to the secret above:

```shell
kubectl patch serviceaccount $SERVICE_ACCOUNT_NAME \
  -p "{\"imagePullSecrets\": [{\"name\": \"registry-credentials\"}]}" -n $NAMESPACE
```

Now, Chains has push permissions for any TaskRuns running under the service account `$SERVICE_ACCOUNT_NAME`.

## Authenticating to Fulcio for Keyless signing

The default [deployment](../config/100-deployment.yaml) will work against
public Fulcio assuming it is installed into an EKS or GKE cluster. You will
just need to add the following to `chains-config` ConfigMap data section in the
`tekton-chains` namespace:

```
  "signers.x509.fulcio.enabled": "true"
```

### Specifying a custom Fulcio endpoint

If you are running your own instance of Fulcio, you need to further
configure Fulcio for this. You need to additionally point Chains to your
fulcio instance by adding this to `chains-config`. In this case, it's a local
k8s service, but you will need to change the URL to point to your Fulcio
instance.

```
  "signers.x509.fulcio.address": "http://fulcio.fulcio-system.svc"
```

### Specifying Spiffe as authentication provider

If you are using Spiffe to authenticate to Fulcio, you will need to configure
your Chains Deployment to fetch the SVID from the Spire agent. This requires
mounting the Agent socket, specifying an environmental variable (if not using
the default of `/tmp/spire-agent/public/api.sock`).

For VolumeMount, replace the k8s SA token, or add if you use it for something
else the following to `tekton-chains-controller` container volumeMounts section:

```
        - name: spiffe-workload-api
          mountPath: /run/spire/sockets/agent.sock
          readOnly: true
```

Specify (if necessary) the non-default Agent socket, by adding the following
to the `tekton-chains-controller` env section:

```
        - name: SPIFFE_ENDPOINT_SOCKET
          value: "/run/spire/sockets/agent.sock"
```

And finally, adding the volume for the Spiffe workload API by adding this
to deployment `volumes` section:

```
      - name: spiffe-workload-api
        hostPath:
          path: /run/spire/sockets/agent.sock
```
