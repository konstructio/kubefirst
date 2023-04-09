/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vultr

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/vultr/govultr/v3"
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

func (c *VultrConfiguration) TestDomainLiveness(dryRun bool, domainName string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestDomainZoneLiveness skipped.")
		return true
	}

	vultrRecordName := "kubefirst-liveness"
	vultrRecordValue := "domain record propagated"

	vultrRecordConfig := &govultr.DomainRecordReq{
		Name:     vultrRecordName,
		Type:     "TXT",
		Data:     vultrRecordValue,
		TTL:      600,
		Priority: govultr.IntToIntPtr(100),
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("domainName %s", domainName)

	//check for existing records
	records, err := c.GetDNSRecords(domainName)
	if err != nil {
		log.Error().Msgf("error getting vultr dns records for domain %s: %s", domainName, err)
		return false
	}

	for _, record := range records {
		if record.Type == "TXT" && record.Name == vultrRecordName {
			return true
		}
	}

	//create record if it does not exist
	_, _, err = c.Client.DomainRecord.Create(c.Context, domainName, vultrRecordConfig)
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

		log.Info().Msgf("%s", vultrRecordName)
		ips, err := net.LookupTXT(fmt.Sprintf("%s.%s", vultrRecordName, domainName))
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), vultrRecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", vultrRecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", vultrRecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve domain dns record. please check your domain registrar")
		}
	}
	return true
}

// GetStorageBuckets retrieves all Vultr object storage buckets
func (c *VultrConfiguration) GetDNSRecords(domainName string) ([]govultr.DomainRecord, error) {
	records, _, _, err := c.Client.DomainRecord.List(c.Context, domainName, &govultr.ListOptions{})
	if err != nil {
		log.Error().Msgf("error getting vultr dns records for domain %s: %s", domainName, err)
		return []govultr.DomainRecord{}, err
	}

	return records, nil
}

// GetDNSInfo determines whether or not a domain exists within Vultr
func (c *VultrConfiguration) GetDNSInfo(domainName string) (string, error) {
	log.Info().Msg("GetDNSInfo (working...)")

	vultrDNSDomain, _, err := c.Client.Domain.Get(c.Context, domainName)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	return vultrDNSDomain.Domain, nil
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
