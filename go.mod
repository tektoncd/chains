module github.com/tektoncd/chains

go 1.13

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.12.0
	github.com/go-openapi/analysis v0.20.1 // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/google/certificate-transparency-go v1.1.1 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/google/go-containerregistry v0.5.0
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210129212729-5c4818de4025
	github.com/hashicorp/go-multierror v1.1.1
	github.com/jedisct1/go-minisign v0.0.0-20210414164026-819d7e2534ac // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/pelletier/go-toml v1.9.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sigstore/cosign v0.3.2-0.20210504221908-6a2c836159a9 // indirect
	github.com/sigstore/sigstore v0.0.0-20210427115853-11e6eaab7cdc
	github.com/spf13/afero v1.6.0 // indirect
	github.com/tektoncd/pipeline v0.23.0
	github.com/tektoncd/plumbing v0.0.0-20210420200944-17170d5e7bc9
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b
	golang.org/x/net v0.0.0-20210423184538-5f58ad60dda6 // indirect
	golang.org/x/oauth2 v0.0.0-20210413134643-5e61552d6c78 // indirect
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	golang.org/x/term v0.0.0-20210422114643-f5beecf764ed // indirect
	google.golang.org/genproto v0.0.0-20210426193834-eac7f76ac494 // indirect
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	knative.dev/pkg v0.0.0-20210127163530-0d31134d5f4e

)

// Knative deps (release-0.20)
replace (
	contrib.go.opencensus.io/exporter/stackdriver => contrib.go.opencensus.io/exporter/stackdriver v0.13.4
	github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v38.2.0+incompatible
)
