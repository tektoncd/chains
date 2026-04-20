# Performance

Tekton Chains exposes a few parameters that can be used to fine tune the controllers execution to
improve its performance as needed.

The controller accepts the following parameters:

`--threads-per-controller` controls the number of concurrent threads the Chains controller
processes. The default value is 2.

`--kube-api-burst` controle the maximum burst for throttle. The default value is 10.

`--kube-api-qps` controles the maximum QPS to the server from the client. The default value is 5.

Modify the `Deployment` to use those parameters, for example:

```yaml
spec:
    template:
        spec:
            containers:
                - image: ghcr.io/tektoncd/chains/controller-92006fd957c0afd31de6a40b3e33b39f:v0.26.2
                  args:
                    - --threads-per-controller=32
                    - --kube-api-burst=100
                    - --kube-api-qps=50
```
