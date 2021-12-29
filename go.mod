module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.98.0
	cloud.google.com/go/storage v1.18.2
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.42.0
	github.com/google/addlicense v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.7.1-0.20211118220127-abdc633f8305
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20211118220127-abdc633f8305
	github.com/google/go-licenses v0.0.0-20210816172045-3099c18c36e1
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.3.0
	github.com/in-toto/in-toto-golang v0.4.0-prerelease
	github.com/mitchellh/mapstructure v1.4.2
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/ryanuber/go-glob v1.0.0
	github.com/secure-systems-lab/go-securesystemslib v0.1.0
	github.com/sigstore/cosign v1.3.2-0.20211124224132-6fc942badabf
	github.com/sigstore/fulcio v0.1.2-0.20210831152525-42f7422734bb
	github.com/sigstore/rekor v0.3.1-0.20211117161348-09070aa96aef
	github.com/sigstore/sigstore v1.0.2-0.20211115214857-534e133ebf9d
	github.com/tektoncd/pipeline v0.27.1-0.20210830150214-8afd1563782d
	github.com/tektoncd/plumbing v0.0.0-20210902122415-a65b22d5f63b
	go.uber.org/zap v1.19.1
	gocloud.dev v0.24.0
	golang.org/x/crypto v0.0.0-20210920023735-84f357641f63
	google.golang.org/api v0.61.0
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1 // indirect
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	k8s.io/code-generator v0.22.1
	knative.dev/pkg v0.0.0-20211216142117-79271798f696
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v55.0.0+incompatible
	github.com/prometheus/common => github.com/prometheus/common v0.26.0
	// k8s.io/api => k8s.io/api v0.21.4
	// k8s.io/apimachinery => k8s.io/apimachinery v0.21.4
	// k8s.io/client-go => k8s.io/client-go v0.21.4
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7
)
