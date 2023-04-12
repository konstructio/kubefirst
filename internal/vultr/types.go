/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"

	"github.com/vultr/govultr/v3"
)

type VultrConfiguration struct {
	Client  *govultr.Client
	Context context.Context
}

type VultrBucketCredentials struct {
	AccessKey       string
	SecretAccessKey string
	Endpoint        string
}

type ServiceAlerts struct {
	ServiceAlerts []ServiceAlert `json:"service_alerts"`
}
type ServiceAlert struct {
	Id        string              `json:"id"`
	Region    string              `json:"region"`
	Subject   string              `json:"subject"`
	StartDate string              `json:"start_date"`
	UpdatedAt string              `json:"updated_at"`
	Status    string              `json:"status"`
	Entries   []ServiceAlertEntry `json:"entries"`
}

type ServiceAlertEntry struct {
	UpdatedAt string `json:"updated_at"`
	Message   string `json:"message"`
}
