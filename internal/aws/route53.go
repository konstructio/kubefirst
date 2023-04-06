/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
)

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

// TestHostedZoneLiveness checks Route53 for the liveness test record
func (conf *AWSConfiguration) TestHostedZoneLiveness(dryRun bool, hostedZoneName string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestHostedZoneLiveness skipped.")
		return true
	}

	route53RecordName := fmt.Sprintf("kubefirst-liveness.%s", hostedZoneName)
	route53RecordValue := "domain record propagated"

	route53Client := route53.NewFromConfig(conf.Config)

	hostedZoneID, err := conf.GetHostedZoneID(hostedZoneName)
	if err != nil {
		log.Error().Msg(err.Error())
	}

	log.Info().Msgf("checking to see if record %s exists", route53RecordName)
	log.Info().Msgf("hostedZoneId %s", hostedZoneID)
	log.Info().Msgf("route53RecordName %s", route53RecordName)

	// check for existing record
	records, err := route53Client.ListResourceRecordSets(context.Background(), &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
	})
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	for _, r := range records.ResourceRecordSets {
		if *r.Name == fmt.Sprintf("%s.", route53RecordName) {
			log.Info().Msg("domain record found")
			return true
		}
	}

	// create record if it does not exist
	record, err := route53Client.ChangeResourceRecordSets(
		context.Background(),
		&route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53Types.ChangeBatch{
				Changes: []route53Types.Change{
					{
						Action: "UPSERT",
						ResourceRecordSet: &route53Types.ResourceRecordSet{
							Name: aws.String(route53RecordName),
							Type: "TXT",
							ResourceRecords: []route53Types.ResourceRecord{
								{
									Value: aws.String(strconv.Quote(route53RecordValue)),
								},
							},
							TTL:           aws.Int64(10),
							Weight:        aws.Int64(100),
							SetIdentifier: aws.String("CREATE liveness check for kubefirst installation"),
						},
					},
				},
				Comment: aws.String("CREATE liveness check for kubefirst installation"),
			},
			HostedZoneId: aws.String(hostedZoneID),
		})
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msgf("record creation status is %s", record.ChangeInfo.Status)

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", route53RecordName)
		ips, err := net.LookupTXT(route53RecordName)
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), route53RecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", route53RecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", route53RecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve hosted zone dns record. please check your domain registrar")
		}
	}
	return true
}

// GetHostedZoneID returns the ID of a hosted zone if valid
func (conf *AWSConfiguration) GetHostedZoneID(hostedZoneName string) (string, error) {
	route53Client := route53.NewFromConfig(conf.Config)
	hostedZones, err := route53Client.ListHostedZonesByName(
		context.Background(),
		&route53.ListHostedZonesByNameInput{
			DNSName: &hostedZoneName,
		},
	)
	if err != nil {
		log.Error().Msg(err.Error())
	}

	var hostedZoneId string

	for _, zone := range hostedZones.HostedZones {
		if *zone.Name == fmt.Sprintf(`%s%s`, hostedZoneName, ".") {
			hostedZoneId = strings.Split(*zone.Id, "/")[2]
		}
	}

	if hostedZoneId == "" {
		return "", fmt.Errorf("could not find hosted zone ID for hosted zone %s", hostedZoneName)
	}

	return hostedZoneId, nil
}
