/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"os"

	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

var Conf VultrConfiguration = VultrConfiguration{
	Client:  NewVultr(),
	Context: context.Background(),
}

func NewVultr() *govultr.Client {
	config := &oauth2.Config{}
	ctx := context.Background()
	ts := config.TokenSource(ctx, &oauth2.Token{AccessToken: os.Getenv("VULTR_API_KEY")})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	return vultrClient
}
