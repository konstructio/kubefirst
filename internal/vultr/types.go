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
