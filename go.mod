module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.94.1
	cloud.google.com/go/storage v1.16.1
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.42.0
	github.com/google/addlicense v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.6.0
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210830114045-7e0ed51a7bb1
	github.com/google/go-licenses v0.0.0-20210816172045-3099c18c36e1
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.2.1
	github.com/in-toto/in-toto-golang v0.2.1-0.20210910132023-02b98c8d4e22
	github.com/mitchellh/mapstructure v1.4.1
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/ryanuber/go-glob v1.0.0
	github.com/secure-systems-lab/go-securesystemslib v0.1.0
	github.com/sigstore/cosign v1.1.1-0.20210914204018-152eefb4bbf3
	github.com/sigstore/fulcio v0.1.2-0.20210831152525-42f7422734bb
	github.com/sigstore/rekor v0.3.0
	github.com/sigstore/sigstore v0.0.0-20210729211320-56a91f560f44
	github.com/tektoncd/pipeline v0.27.1-0.20210830150214-8afd1563782d
	github.com/tektoncd/plumbing v0.0.0-20210902122415-a65b22d5f63b
	github.com/theupdateframework/go-tuf v0.0.0-20210804171843-477a5d73800a // indirect
	go.uber.org/zap v1.19.0
	gocloud.dev v0.24.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	google.golang.org/api v0.56.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/code-generator v0.22.1
	k8s.io/utils v0.0.0-20210802155522-efc7438f0176
	knative.dev/pkg v0.0.0-20210908025933-71508fc69a57
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/prometheus/common => github.com/prometheus/common v0.26.0
	k8s.io/api => k8s.io/api v0.21.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
)
