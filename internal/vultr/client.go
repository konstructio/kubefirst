/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

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

// HealthCheck looks for any service alerts in the specified region
func (c *VultrConfiguration) HealthCheck(region string) error {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	var output ServiceAlerts

	resp, err := httpClient.Get("https://status.vultr.com/alerts.json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&output)
	if err != nil {
		return err
	}

	for _, alert := range output.ServiceAlerts {
		if alert.Region == region {
			fmt.Println(alert.Entries)
		}
	}

	return nil
}
