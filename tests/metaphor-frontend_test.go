/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/kubefirst/runtime/configs"
	"github.com/kubefirst/runtime/pkg"
	"github.com/spf13/viper"
)

// TestLocalMetaphorFrontendEndToEnd tests the Metaphor frontend (dev, staging, prod), and look for a http response code of 200
func TestLocalMetaphorFrontendEndToEnd(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
	}

	testCases := []struct {
		name     string
		url      string
		expected int
	}{
		{name: "metaphor frontend development", url: pkg.MetaphorFrontendSlimTLSDev, expected: http.StatusOK},
		{name: "metaphor frontend staging", url: pkg.MetaphorFrontendSlimTLSStaging, expected: http.StatusOK},
		{name: "metaphor frontend production", url: pkg.MetaphorFrontendSlimTLSProd, expected: http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			resp, err := http.Get(tc.url)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			defer resp.Body.Close()

			fmt.Println("HTTP status code:", resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("HTTP status code is not 200")
			}
		})
	}

}

// TestCloudMetaphorsEndToEnd tests the Metaphor frontend, and look for a http response code of 200 for cloud
func TestCloudMetaphorsEndToEnd(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping end to tend test")
	}

	config := configs.ReadConfig()
	if err := pkg.SetupViper(config); err != nil {
		t.Errorf(err.Error())
	}

	testCases := []struct {
		name     string
		url      string
		expected int
	}{
		{name: "metaphor frontend development", url: "https://metaphor-development." + viper.GetString("aws.hostedzonename"), expected: http.StatusOK},
		{name: "metaphor frontend staging", url: "https://metaphor-staging." + viper.GetString("aws.hostedzonename"), expected: http.StatusOK},
		{name: "metaphor frontend production", url: "https://metaphor-production." + viper.GetString("aws.hostedzonename"), expected: http.StatusOK},
		{name: "metaphor NodeJs development", url: "https://metaphor-development." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
		{name: "metaphor NodeJs staging", url: "https://metaphor-staging." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
		{name: "metaphor NodeJs production", url: "https://metaphor-production." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
		{name: "metaphor Go development", url: "https://metaphor-go-development." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
		{name: "metaphor Go staging", url: "https://metaphor-go-staging." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
		{name: "metaphor Go production", url: "https://metaphor-go-production." + viper.GetString("aws.hostedzonename") + "/app", expected: http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			resp, err := http.Get(tc.url)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			defer resp.Body.Close()

			fmt.Println("HTTP status code:", resp.StatusCode)

			if resp.StatusCode != http.StatusOK {
				t.Errorf("HTTP status code is not 200")
			}
			if resp.StatusCode != tc.expected {
				t.Errorf("[%s] wanted http status code (%d), got (%d)", tc.url, resp.StatusCode, tc.expected)
			}
		})
	}

}
