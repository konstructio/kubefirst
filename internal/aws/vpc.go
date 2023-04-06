/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rs/zerolog/log"
)

const (
	// minimumAvailabilityZones is a hard limit on the amount of availability
	// zones a region must have to proceed with install
	minimumAvailabilityZones int = 3
)

// CheckAvailabilityZones determines whether or not an aws region is compatible
// with the minimum availability zone requirement specified by consumption of
// aws Terraform modules
func (conf *AWSConfiguration) CheckAvailabilityZones(region string) (bool, error) {
	ec2Client := ec2.NewFromConfig(conf.Config, func(o *ec2.Options) {
		o.Region = region
	})

	availabilityZones, err := ec2Client.DescribeAvailabilityZones(
		context.Background(),
		&ec2.DescribeAvailabilityZonesInput{},
	)
	if err != nil {
		return false, err
	}

	numAavailabilityZones := len(availabilityZones.AvailabilityZones)
	log.Info().Msgf("aws region %s has %v availability zones", region, numAavailabilityZones)
	for _, az := range availabilityZones.AvailabilityZones {
		log.Info().Msg(*az.ZoneName)
	}

	if numAavailabilityZones < minimumAvailabilityZones {
		compatibleRegions, err := conf.ListCompatibleRegions()
		if err != nil {
			log.Error().Msgf("error getting valid aws regions - skipping: %s", err)
		}

		return false, fmt.Errorf(
			"aws region %s has %v availability zones - kubefirst requires at least %v - please select a different region\n\nthe following regions are compatible:\n\n%v\n",
			region,
			numAavailabilityZones,
			minimumAvailabilityZones,
			compatibleRegions,
		)
	}

	return true, nil
}

// ListCompatibleRegions returns aws regions that have the minimum number of availability zones
// required to support the kubefirst platform
func (conf *AWSConfiguration) ListCompatibleRegions() ([]string, error) {
	ec2Client := ec2.NewFromConfig(conf.Config)

	regions, err := ec2Client.DescribeRegions(
		context.Background(),
		&ec2.DescribeRegionsInput{},
	)
	if err != nil {
		return []string{}, err
	}

	filterName := "region-name"
	compatibleRegions := make([]string, 0)
	for _, region := range regions.Regions {
		ec2Client := ec2.NewFromConfig(conf.Config, func(o *ec2.Options) {
			o.Region = *region.RegionName
		})
		availabilityZones, err := ec2Client.DescribeAvailabilityZones(
			context.Background(),
			&ec2.DescribeAvailabilityZonesInput{
				Filters: []ec2Types.Filter{
					{
						Name:   &filterName,
						Values: []string{*region.RegionName},
					},
				},
			},
		)
		if err != nil {
			return []string{}, err
		}

		if len(availabilityZones.AvailabilityZones) >= minimumAvailabilityZones {
			compatibleRegions = append(compatibleRegions, *region.RegionName)
		}
	}

	return compatibleRegions, nil
}
