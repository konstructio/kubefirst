package ssl

import (
	"crypto/tls"
	"testing"
)

// todo: use URL constants for app addresses
func TestArgoCertificateIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const SSLPort = ":443"

	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "argo",
			address: "argo.kubefirst.dev",
		},
		{
			name:    "argocd",
			address: "argocd.kubefirst.dev",
		},
		{
			name:    "atlantis",
			address: "atlantis.kubefirst.dev",
		},
		{
			name:    "chartmuseum",
			address: "chartmuseum.kubefirst.dev",
		},
		{
			name:    "vault",
			address: "vault.kubefirst.dev",
		},
		{
			name:    "minio",
			address: "minio.kubefirst.dev",
		},
		{
			name:    "minio-console",
			address: "minio-console.kubefirst.dev",
		},
		{
			name:    "kubefirst",
			address: "kubefirst.kubefirst.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := tls.Dial("tcp", tt.address+SSLPort, nil)
			if err != nil {
				t.Logf("testing %s , address %s", tt.name, tt.address)
				t.Error(err)
				return
			}
			err = conn.VerifyHostname(tt.address)
			if err != nil {
				t.Error(err)
			}
		})
	}

}
