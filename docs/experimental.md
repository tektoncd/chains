<!--
---
linkTitle: "Experimental Features"
weight: 60
---
-->

# Experimental Features

This doc covers experimental features in Tekton Chains.

Currently, experimental features include:

- [PubSub Storage Backend Support](#Pubsub-Storage-Backend-Support)

## PubSub Storage Backend Support

Support for PubSub storage backend was introduced in chains. The first PubSub
provider implementation is Kafka, and more may follow in the future.

### Kafka

To enable the Kafka backend run:

```shell
kubectl patch configmap chains-config -n tekton-chains -p='{"data": {storage.pubsub.provider": "kafka","storage.pubsub.topic": "chains", "storage.pubsub.kafka.bootstrap.servers":"kafka-0.kafka-headless.default.svc.cluster.local:9092"}}'
```

Note that the `storage.pubsub.kafka.bootstrap.servers` value needs to be
adjusted to point to the list of [bootstrap servers] your cluster is connected
to.

[bootstrap servers]: https://kafka.apache.org/documentation/#producerconfigs_bootstrap.servers
