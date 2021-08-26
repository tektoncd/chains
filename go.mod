module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.90.0
	cloud.google.com/go/storage v1.16.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.41.1
	github.com/google/addlicense v0.0.0-20210809195240-d43bb61fdfda
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.6.0
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e
	github.com/google/go-licenses v0.0.0-20210329231322-ce1d9163b77d
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.2.1
	github.com/in-toto/in-toto-golang v0.2.1-0.20210627200632-886210ae2ab9
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/statsd_exporter v0.20.3 // indirect
	github.com/ryanuber/go-glob v1.0.0
	github.com/sigstore/cosign v1.0.2-0.20210824191708-7b08e21cf07f
	github.com/sigstore/fulcio v0.1.1
	github.com/sigstore/rekor v0.3.0
	github.com/sigstore/sigstore v0.0.0-20210729211320-56a91f560f44
	github.com/tektoncd/pipeline v0.27.1-0.20210818181609-67b318ba62d9
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	go.uber.org/zap v1.18.1
	gocloud.dev v0.23.0
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	google.golang.org/api v0.54.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/code-generator v0.20.7
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	knative.dev/pkg v0.0.0-20210818135208-7b5ecbc0e477
	sigs.k8s.io/structured-merge-diff/v4 v4.1.0 // indirect
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/prometheus/common => github.com/prometheus/common v0.26.0
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
)
