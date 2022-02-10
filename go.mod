module github.com/tektoncd/chains

go 1.16

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c

require (
	cloud.google.com/go/compute v1.2.0
	cloud.google.com/go/storage v1.20.0
	github.com/armon/go-metrics v0.3.10
	github.com/armon/go-radix v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.44.0
	github.com/google/addlicense v1.0.0
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220202214207-9c35968ef47e
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220125170349-50dfc2733d10
	github.com/google/go-licenses v0.0.0-20210816172045-3099c18c36e1
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-hclog v1.1.0
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/go-version v1.4.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/vault/sdk v0.3.0
	github.com/in-toto/in-toto-golang v0.3.4-0.20211211042327-af1f9fb822bf
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/mitchellh/mapstructure v1.4.3
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/secure-systems-lab/go-securesystemslib v0.3.0
	github.com/sigstore/cosign v1.5.2-0.20220210140103-2381756282ae
	github.com/sigstore/rekor v0.4.1-0.20220114213500-23f583409af3
	github.com/sigstore/sigstore v1.1.1-0.20220130134424-bae9b66b8442
	github.com/tektoncd/pipeline v0.31.1-0.20220105002759-3e137645be61
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.21.0
	gocloud.dev v0.24.1-0.20211119014450-028788aaaa4c
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/code-generator v0.22.5
	knative.dev/pkg v0.0.0-20220121092305-3ba5d72e310a
)
