/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/civo/civogo"
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

// TestDomainLiveness checks Civo DNS for the liveness test record
func TestDomainLiveness(dryRun bool, domainName, domainId, region string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestDomainZoneLiveness skipped.")
		return true
	}

	civoRecordName := fmt.Sprintf("kubefirst-liveness.%s", domainName)
	civoRecordValue := "domain record propagated"

	civoClient, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return log.Logger.Fatal().Stack().Enabled()
	}

	civoRecordConfig := &civogo.DNSRecordConfig{
		Type:     civogo.DNSRecordTypeTXT,
		Name:     civoRecordName,
		Value:    civoRecordValue,
		Priority: 100,
		TTL:      600,
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("domainId %s", domainId)
	log.Info().Msgf("domainName %s", domainName)

	//check for existing records
	records, err := civoClient.ListDNSRecords(domainId)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	if len(records) > 0 {
		log.Info().Msg("domain record found")
		return true
	}

	//create record if it does not exist
	_, err = civoClient.CreateDNSRecord(domainId, civoRecordConfig)
	if err != nil {
		log.Warn().Msgf("%s", err)
		return false
	}
	log.Info().Msg("domain record created")

	count := 0
	// todo need to exit after n number of minutes and tell them to check ns records
	// todo this logic sucks
	for count <= 100 {
		count++

		log.Info().Msgf("%s", civoRecordName)
		ips, err := net.LookupTXT(civoRecordName)
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), civoRecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", civoRecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", civoRecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve domain dns record. please check your domain registrar")
		}
	}
	return true
}

// GetDomainApexContent determines whether or not a target domain features
// a host responding at zone apex
func GetDomainApexContent(domainName string) bool {
	timeout := time.Duration(5 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}

	exists := false
	for _, proto := range []string{"http", "https"} {
		fqdn := fmt.Sprintf("%s://%s", proto, domainName)
		_, err := client.Get(fqdn)
		if err != nil {
			log.Warn().Msgf("domain %s has no apex content", fqdn)
		} else {
			log.Info().Msgf("domain %s has apex content", fqdn)
			exists = true
		}
	}

	return exists
}

// GetDNSInfo try to reach the provided domain
func GetDNSInfo(domainName, region string) (string, error) {

	log.Info().Msg("GetDNSInfo (working...)")

	civoClient, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	civoDNSDomain, err := civoClient.FindDNSDomain(domainName)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	return civoDNSDomain.ID, nil

}
