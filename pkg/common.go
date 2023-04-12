/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"net/http"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var SupportedPlatforms = []string{
	"aws-github",
	"aws-gitlab",
	"civo-github",
	"civo-gitlab",
	"digitalocean-github",
	"k3d-github",
	"k3d-gitlab",
	"vultr-github",
	"vultr-gitlab",
}
