# Metrics

Tekton Chains exposes standard
[Knative Controller metrics](https://knative.dev/docs/serving/observability/metrics/serving-metrics/#controller).
These metrics are served on `/metrics` on the Tekton Chains controller Pod.
These are exposed on port `:9090` by default.

Metric collectors like Prometheus and OpenTelemetry can be used to collect these
metrics. See
[Knative - Collecting Metrics](https://knative.dev/docs/serving/observability/metrics/collecting-metrics/)
for more details.
