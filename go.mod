module github.com/tektoncd/chains

go 1.13

require (
	cloud.google.com/go v0.72.0
	cloud.google.com/go/storage v1.10.0
	github.com/Azure/go-autorest/autorest v0.10.2 // indirect
	github.com/google/go-cmp v0.5.4
	github.com/google/go-containerregistry v0.4.1-0.20210128200529-19c2b639fab1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210129212729-5c4818de4025
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.23.0
	github.com/tektoncd/plumbing v0.0.0-20210420200944-17170d5e7bc9
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	knative.dev/pkg v0.0.0-20210127163530-0d31134d5f4e

)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)
