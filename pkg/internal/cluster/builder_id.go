package cluster

import (
	"context"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"knative.dev/pkg/logging"
)

var (
	provider = &tokenSource{
		path: "/var/run/secrets/kubernetes.io/serviceaccount/token",
	}
)

type tokenSource struct {
	path string

	token *oauth2.Token
	iss   string
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	if ts.token.Valid() {
		return ts.token, nil
	}
	b, err := os.ReadFile(ts.path)
	if err != nil {
		return nil, err
	}
	ts.token = &oauth2.Token{
		AccessToken: string(b),
	}

	return ts.token, nil
}

func (ts *tokenSource) Issuer(ctx context.Context) string {
	log := logging.FromContext(ctx)

	if ts.token.Valid() && ts.iss != "" {
		return ts.iss
	}

	oauth, err := ts.Token()
	if err != nil {
		log.Errorf("failed to get cluster token: %v", err)
		return ""
	}

	// We're assuming that in order to place a token in the path
	// you already have some amount of privilege.
	// While we don't know if the token is actually real, even if we
	// wanted to verify it against the api server we'd need to trust
	// the cacert bundle also included in this same path, so trusting
	// the token is correctly set by the Kubernetes cluster is
	// likely a reasonable compromise.
	parser := jwt.NewParser()
	claims := new(jwt.RegisteredClaims)
	if _, _, err := parser.ParseUnverified(oauth.AccessToken, claims); err != nil {
		log.Errorf("failed to parse cluster token: %v", err)
		return ""
	}

	ts.iss = claims.Issuer
	return claims.Issuer
}

// Cluster returns an ID that identifies the current cluster.
// To approximate this, we use the cluster token issuer as defined by
// the controller's default service account token.
// See https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api for more details.
// This will be change depending on where/how the cluster is running.
//
// Some examples:
// - GKE: https://containers.googleapis.com/v1/projects/123456789012/locations/us-east1/clusters/cluster-1
// - EKS: https://oidc.eks.us-east-1.amazonaws.com/id/12345678901234567890123456789012
// - AKS: https://eastus.oic.prod-aks.azure.com/00000000-0000-0000-0000-000000000000/00000000-0000-0000-0000-000000000000/
// - Kind/Local: https://kubernetes.default.svc (NOTE: this isn't a real URL and won't give you much useful information)
func ClusterID(ctx context.Context) string {
	return provider.Issuer(ctx)
}
