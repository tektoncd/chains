# Metrics

Tekton Chains exposes standard
[Knative Controller metrics](https://knative.dev/docs/serving/observability/metrics/serving-metrics/#controller).
These metrics are served on `/metrics` on the Tekton Chains controller Pod.
These are exposed on port `:9090` by default.

Metric collectors like Prometheus and OpenTelemetry can be used to collect these
metrics. See
[Knative - Collecting Metrics](https://knative.dev/docs/serving/observability/metrics/collecting-metrics/)
for more details.

# Chains Controller Metrics

The following chains metrics are also available at `tekton-chains-metrics` service on port `9090`.

| Name                                                                                    | Type | Description |
|-----------------------------------------------------------------------------------------| ----------- | ----------- |
| `watcher_pipelinerun_sign_created_total`                                             | Counter | Total number of signed messages for pipelineruns |
| `watcher_pipelinerun_payload_uploaded_total`                                             | Counter | Total number of uploaded payloads for pipelineruns |
| `watcher_pipelinerun_payload_stored_total`                                             | Counter | Total number of stored payloads for pipelineruns |
| `watcher_pipelinerun_marked_signed_total`                                             | Counter | Total number of objects marked as signed for pipelineruns |
| `watcher_taskrun_sign_created_total`                                             | Counter | Total number of signed messages for taskruns |
| `watcher_taskrun_payload_uploaded_total`                                             | Counter | Total number of uploaded payloads for taskruns |
| `watcher_taskrun_payload_stored_total`                                             | Counter | Total number of stored payloads for taskruns |
| `watcher_taskrun_marked_signed_total`                                             | Counter | Total number of objects marked as signed for taskruns |

To access the chains metrics, use the following commands:
```shell
kubectl port-forward -n tekton-chains service/tekton-chains-metrics 9090
```

And then check that changes have been applied to metrics coming from [http://127.0.0.1:9090/metrics](http://127.0.0.1:9090/metrics)
