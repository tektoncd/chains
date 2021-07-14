module github.com/tektoncd/chains

go 1.16

require (
	cloud.google.com/go v0.86.0
	cloud.google.com/go/storage v1.15.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/strfmt v0.20.1
	github.com/go-openapi/swag v0.19.15
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e
	github.com/hashicorp/go-multierror v1.1.1
	github.com/in-toto/in-toto-golang v0.2.1-0.20210627200632-886210ae2ab9
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.11.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/statsd_exporter v0.20.3 // indirect
	github.com/sigstore/cosign v0.5.1-0.20210710010214-577485d73df9
	github.com/sigstore/rekor v0.2.1-0.20210709135953-5862799208a2
	github.com/sigstore/sigstore v0.0.0-20210709190449-2ab5ec881a5f
	github.com/tektoncd/pipeline v0.25.0
	github.com/tektoncd/plumbing v0.0.0-20210514044347-f8a9689d5bd5
	go.uber.org/zap v1.18.1
	gocloud.dev v0.23.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
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
