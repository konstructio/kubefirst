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
			address: "argo.localdev.me",
		},
		{
			name:    "argocd",
			address: "argocd.localdev.me",
		},
		{
			name:    "atlantis",
			address: "atlantis.localdev.me",
		},
		{
			name:    "chartmuseum",
			address: "chartmuseum.localdev.me",
		},
		{
			name:    "vault",
			address: "vault.localdev.me",
		},
		{
			name:    "minio",
			address: "minio.localdev.me",
		},
		{
			name:    "minio-console",
			address: "minio-console.localdev.me",
		},
		{
			name:    "kubefirst",
			address: "kubefirst.localdev.me",
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
