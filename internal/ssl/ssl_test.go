package ssl

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/kubefirst/kubefirst/internal/k3d"
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
			address: fmt.Sprintf("argo.%s", k3d.DomainName),
		},
		{
			name:    "argocd",
			address: fmt.Sprintf("argocd.%s", k3d.DomainName),
		},
		{
			name:    "atlantis",
			address: fmt.Sprintf("atlantis.%s", k3d.DomainName),
		},
		{
			name:    "chartmuseum",
			address: fmt.Sprintf("chartmuseum.%s", k3d.DomainName),
		},
		{
			name:    "vault",
			address: fmt.Sprintf("vault.%s", k3d.DomainName),
		},
		{
			name:    "minio",
			address: fmt.Sprintf("minio.%s", k3d.DomainName),
		},
		{
			name:    "minio-console",
			address: fmt.Sprintf("minio-console.%s", k3d.DomainName),
		},
		{
			name:    "kubefirst",
			address: fmt.Sprintf("kubefirst.%s", k3d.DomainName),
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
