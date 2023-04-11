/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package httpCommon

import (
	"crypto/tls"
	"net/http"
	"time"
)

// CustomHttpClient - creates a http client based on k1 standards
// allowInsecure defines: tls.Config{InsecureSkipVerify: allowInsecure}
func CustomHttpClient(allowInsecure bool) *http.Client {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: allowInsecure}
	httpClient := http.Client{
		Transport: customTransport,
		Timeout:   time.Second * 90,
	}
	return &httpClient
}

// ResolveAddress returns whether or not an address is resolvable
func ResolveAddress(address string) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}

	_, err := httpClient.Get(address)
	if err != nil {
		return err
	}

	return nil
}
