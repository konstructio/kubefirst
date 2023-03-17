package helpers

import (
	"crypto/tls"
	"fmt"
)

// TestEndpointTLS determines whether or not an endpoint accepts connections over https
func TestEndpointTLS(endpoint string) error {
	_, err := tls.Dial("tcp", fmt.Sprintf("%s:443", endpoint), nil)
	if err != nil {
		return fmt.Errorf("endpoint %s doesn't support tls: %s", endpoint, err)
	}

	return nil
}
