package cluster

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestBuilderID(t *testing.T) {
	// Create the Claims
	claims := &jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		Issuer:    "example.com",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ss, err := token.SignedString(pk)
	if err != nil {
		t.Fatal(err)
	}

	// This is a real token taken from a local kind cluster.
	realToken, err := os.ReadFile("testdata/token.txt")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		token []byte
		want  string
	}{
		{
			token: []byte(ss),
			want:  claims.Issuer,
		},
		{
			token: realToken,
			want:  "https://kubernetes.default.svc",
		},
	} {
		t.Run(tc.want, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "token")
			if err := os.WriteFile(path, []byte(ss), 0600); err != nil {
				t.Fatal(err)
			}

			provider = &tokenSource{
				path: path,
			}

			ctx := logtesting.TestContextWithLogger(t)
			got := ClusterID(ctx)
			if got != claims.Issuer {
				t.Errorf("BuilderID() = %s, want %s", got, claims.Issuer)
			}
		})
	}
}
