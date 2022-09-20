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
