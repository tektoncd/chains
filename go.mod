module github.com/tektoncd/chains

go 1.16

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c

require (
	cloud.google.com/go/compute v1.5.0
	cloud.google.com/go/storage v1.21.0
	github.com/armon/go-metrics v0.3.10
	github.com/armon/go-radix v1.0.0
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/golang/snappy v0.0.4
	github.com/golangci/golangci-lint v1.44.2
	github.com/google/addlicense v1.0.0
	github.com/google/go-cmp v0.5.7
	github.com/google/go-containerregistry v0.8.1-0.20220216220642-00c59d91847c
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20220310143843-f1fa40b162a1
	github.com/google/go-licenses v0.0.0-20210816172045-3099c18c36e1
	github.com/grafeas/grafeas v0.2.1
	github.com/hashicorp/errwrap v1.1.0
	github.com/hashicorp/go-hclog v1.2.0
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
	github.com/hashicorp/vault/sdk v0.4.1
	github.com/in-toto/in-toto-golang v0.3.4-0.20211211042327-af1f9fb822bf
	github.com/letsencrypt/boulder v0.0.0-20220331220046-b23ab962616e
	github.com/mitchellh/copystructure v1.2.0
	github.com/mitchellh/go-testing-interface v1.14.1
	github.com/mitchellh/mapstructure v1.4.3
	github.com/pierrec/lz4 v2.6.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/secure-systems-lab/go-securesystemslib v0.3.1
	github.com/sigstore/cosign v1.6.1-0.20220401112351-80034ba3b905
	github.com/sigstore/rekor v0.4.1-0.20220114213500-23f583409af3
	github.com/sigstore/sigstore v1.2.1-0.20220330193110-d7475aecf1db
	github.com/tektoncd/pipeline v0.31.1-0.20220105002759-3e137645be61
	github.com/tektoncd/plumbing v0.0.0-20211012143332-c7cc43d9bc0c
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399
	go.uber.org/atomic v1.9.0
	go.uber.org/zap v1.21.0
	gocloud.dev v0.24.1-0.20211119014450-028788aaaa4c
	golang.org/x/crypto v0.0.0-20220214200702-86341886e292
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.23.5
	k8s.io/client-go v0.23.5
	k8s.io/code-generator v0.23.5
	knative.dev/pkg v0.0.0-20220329144915-0a1ec2e0d46c
)
