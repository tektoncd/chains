module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.15.0
	github.com/ghodss/yaml v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e
	github.com/hashicorp/go-multierror v1.1.1
	github.com/in-toto/in-toto-golang v0.1.1-0.20210505200736-471bd79ebd18
	github.com/pkg/errors v0.9.1
	github.com/sigstore/cosign v0.5.0
	github.com/sigstore/sigstore v0.0.0-20210530211317-99216b8b86a6
	github.com/tektoncd/pipeline v0.23.0
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	go.uber.org/zap v1.16.0
	gocloud.dev v0.23.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.19.7
	knative.dev/pkg v0.0.0-20210127163530-0d31134d5f4e

)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)
