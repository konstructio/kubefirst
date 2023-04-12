/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/digitalocean/godo"
)

var Conf DigitaloceanConfiguration = DigitaloceanConfiguration{
	Client:  NewDigitalocean(),
	Context: context.Background(),
}

func NewDigitalocean() *godo.Client {
	digitaloceanClient := godo.NewFromToken(os.Getenv("DO_TOKEN"))

	return digitaloceanClient
}

// ValidateRegion guarantees a region argument is valid
func (c *DigitaloceanConfiguration) ValidateRegion(region string) error {
	regions, _, err := c.Client.Regions.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return err
	}

	regionSlugs := make([]string, 0)
	for _, region := range regions {
		regionSlugs = append(regionSlugs, region.Slug)
	}
	sort.Strings(regionSlugs)
	validRegion := false
	for _, rsl := range regionSlugs {
		if rsl == region {
			validRegion = true
		}
	}
	if !validRegion {
		return fmt.Errorf("%s is not a valid region option - please use one of: %v", region, regionSlugs)
	}

	// Regions where spaces are enabled
	regionsWithSpaces := []string{"ams3", "fra1", "nyc3", "sfo3", "sgp1", "syd1"}
	sort.Strings(regionsWithSpaces)
	validRegion = false
	for _, rws := range regionsWithSpaces {
		if rws == region {
			validRegion = true
		}
	}
	if !validRegion {
		return fmt.Errorf("while %s is a valid region, it does not support spaces - please use one of: %v", region, regionsWithSpaces)
	}

	return nil
}
