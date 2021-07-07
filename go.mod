module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.15.0
	github.com/GoogleContainerTools/skaffold v1.25.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/analysis v0.20.1 // indirect
	github.com/golangci/golangci-lint v1.40.1 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.6.8 // indirect
	github.com/hashicorp/vault/api v1.1.0 // indirect
	github.com/hashicorp/vault/sdk v0.2.0 // indirect
	github.com/in-toto/in-toto-golang v0.2.1-0.20210627200632-886210ae2ab9
	github.com/jedisct1/go-minisign v0.0.0-20210414164026-819d7e2534ac // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/open-policy-agent/opa v0.30.1 // indirect
	github.com/pierrec/lz4 v2.6.0+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.29.0 // indirect
	github.com/prometheus/statsd_exporter v0.20.3 // indirect
	github.com/sassoftware/relic v0.0.0-20210427151427-dfb082b79b74 // indirect
	github.com/sigstore/cosign v0.5.1-0.20210707123827-04a0cf3ed4ee
	github.com/sigstore/rekor v0.2.1-0.20210705133645-dbbbff597bc2
	github.com/sigstore/sigstore v0.0.0-20210706214059-9f37c836c049
	github.com/tektoncd/pipeline v0.25.0
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	github.com/theupdateframework/go-tuf v0.0.0-20210630170422-22a94818d17b // indirect
	go.mongodb.org/mongo-driver v1.5.3 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.18.1
	gocloud.dev v0.23.0
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	google.golang.org/genproto v0.0.0-20210701191553-46259e63a0a9 // indirect
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210305001622-591a79e4bda7 // indirect
	knative.dev/pkg v0.0.0-20210510175900-4564797bf3b7
	sigs.k8s.io/structured-merge-diff/v4 v4.1.0 // indirect
)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/prometheus/common => github.com/prometheus/common v0.26.0
	k8s.io/api => k8s.io/api v0.20.7
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.7
	k8s.io/client-go => k8s.io/client-go v0.20.7
)
