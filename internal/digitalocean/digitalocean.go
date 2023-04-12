/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/digitalocean/godo"
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

func (c *DigitaloceanConfiguration) TestDomainLiveness(dryRun bool, domainName string) bool {
	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, TestDomainLiveness skipped.")
		return true
	}

	doRecordName := "kubefirst-liveness"
	doRecordValue := "domain record propagated"

	doRecordConfig := &godo.DomainRecordEditRequest{
		Name:     doRecordName,
		Type:     "TXT",
		Data:     doRecordValue,
		TTL:      600,
		Priority: *godo.Int(100),
	}

	log.Info().Msgf("checking to see if record %s exists", domainName)
	log.Info().Msgf("domainName %s", domainName)

	//check for existing records
	records, err := c.GetDNSRecords(domainName)
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return false
	}

	for _, record := range records {
		if record.Type == "TXT" && record.Name == doRecordName {
			return true
		}
	}

	//create record if it does not exist
	_, _, err = c.Client.Domains.CreateRecord(c.Context, domainName, doRecordConfig)
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

		log.Info().Msgf("%s", doRecordName)
		ips, err := net.LookupTXT(fmt.Sprintf("%s.%s", doRecordName, domainName))
		if err != nil {
			ips, err = backupResolver.LookupTXT(context.Background(), doRecordName)
		}

		log.Info().Msgf("%s", ips)

		if err != nil {
			log.Warn().Msgf("Could not get record name %s - waiting 10 seconds and trying again: \nerror: %s", doRecordName, err)
			time.Sleep(10 * time.Second)
		} else {
			for _, ip := range ips {
				// todo check ip against route53RecordValue in some capacity so we can pivot the value for testing
				log.Info().Msgf("%s. in TXT record value: %s\n", doRecordName, ip)
				count = 101
			}
		}
		if count == 100 {
			log.Panic().Msg("unable to resolve domain dns record. please check your domain registrar")
		}
	}
	return true
}

// GetDNSRecords retrieves DNS records
func (c *DigitaloceanConfiguration) GetDNSRecords(domainName string) ([]godo.DomainRecord, error) {
	records, _, err := c.Client.Domains.Records(c.Context, domainName, &godo.ListOptions{})
	if err != nil {
		log.Error().Msgf("error getting digitalocean dns records for domain %s: %s", domainName, err)
		return []godo.DomainRecord{}, err
	}

	return records, nil
}

// GetDNSInfo determines whether or not a domain exists within digitalocean
func (c *DigitaloceanConfiguration) GetDNSInfo(domainName string) (string, error) {
	log.Info().Msg("GetDNSInfo (working...)")

	doDNSDomain, _, err := c.Client.Domains.Get(c.Context, domainName)
	if err != nil {
		log.Info().Msg(err.Error())
		return "", err
	}

	return doDNSDomain.Name, nil
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
