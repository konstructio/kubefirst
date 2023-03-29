package aws

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// TXTRecord stores Route53 TXT record data
type TXTRecord struct {
	Name          string
	Value         string
	SetIdentifier *string
	Weight        *int64
	TTL           int64
}

// ARecord stores Route53 A record data
type ARecord struct {
	Name        string
	RecordType  string
	TTL         *int64
	AliasTarget *route53Types.AliasTarget
}

// Some systems fail to resolve TXT records, so try to use Google as a backup
var backupResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{
			Timeout: time.Millisecond * time.Duration(10000),
		}
		return d.DialContext(ctx, network, "8.8.8.8:53")
	},
}

func NewAwsV2(region string) aws.Config {
	// todo these should also be supported flags
	profile := os.Getenv("AWS_PROFILE")

	awsClient, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		log.Panic().Msg("unable to create aws client")
	}

	return awsClient
}
