module github.com/tektoncd/chains

go 1.14

require (
	cloud.google.com/go v0.62.0
	cloud.google.com/go/storage v1.10.0
	contrib.go.opencensus.io/exporter/ocagent v0.7.0 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.2.0 // indirect
	github.com/google/go-cmp v0.5.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/tektoncd/pipeline v0.14.2
	github.com/tektoncd/plumbing v0.0.0-20200615101855-f56d60092af6
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	golang.org/x/sys v0.0.0-20200805065543-0cf7623e9dbd // indirect
	golang.org/x/tools v0.0.0-20200804234916-fec4f28ebb08 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/genproto v0.0.0-20200804151602-45615f50871c // indirect
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20200616145124-d289d2eef6cb
)

// Knative deps (release-0.15)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.4.0+incompatible
	knative.dev/caching => knative.dev/caching v0.0.0-20200521155757-e78d17bc250e
	knative.dev/pkg => knative.dev/pkg v0.0.0-20200528142800-1c6815d7e4c9
)

// Pin k8s deps to 1.16.5
replace (
	k8s.io/api => k8s.io/api v0.16.5
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.5
	k8s.io/client-go => k8s.io/client-go v0.16.5
	k8s.io/code-generator => k8s.io/code-generator v0.16.5
)
