module github.com/tektoncd/chains

go 1.16

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c

require (
	cloud.google.com/go v0.99.0
	cloud.google.com/go/storage v1.18.2
	github.com/armon/go-metrics v0.3.10
	github.com/armon/go-radix v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.43.0
	github.com/google/addlicense v1.0.0
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.7.1-0.20211203164431-c75901cce627
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20211215180950-ab77ea68f600
	github.com/google/go-licenses v0.0.0-20210816172045-3099c18c36e1
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-hclog v1.0.0
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-plugin v1.4.3
	github.com/hashicorp/go-secure-stdlib/mlock v0.1.2
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/go-version v1.3.0
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
	github.com/sigstore/cosign v1.4.2-0.20220103014340-1a7f9d61a2e9
	github.com/sigstore/fulcio v0.1.2-0.20220103193424-0df42390d392 // indirect
	github.com/sigstore/rekor v0.4.1-0.20220103184137-e86cf37242d3
	github.com/sigstore/sigstore v1.1.1-0.20220104191147-28bc731b4695
	github.com/tektoncd/pipeline v0.31.1-0.20220105002759-3e137645be61
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.19.1
	gocloud.dev v0.24.1-0.20211119014450-028788aaaa4c
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.23.2
	k8s.io/apimachinery v0.23.2
	k8s.io/client-go v0.23.2
	k8s.io/code-generator v0.22.5
	knative.dev/pkg v0.0.0-20220104185830-52e42b760b54
)
