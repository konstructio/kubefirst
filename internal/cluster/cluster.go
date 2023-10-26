/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package cluster

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rs/zerolog/log"

	apiTypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/types"
)

func GetConsoleIngresUrl() string {

	if strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true" { //allow using local console running on port 3000
		return "http://localhost:3000"
	}

	return "https://console.kubefirst.dev"
}

func CreateCluster(cluster apiTypes.ClusterDefinition) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyCreateClusterRequest{
		Body: cluster,
		Url:  fmt.Sprintf("/cluster/%s", cluster.ClusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", GetConsoleIngresUrl()), bytes.NewReader(payload))
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Info().Msgf("unable to create cluster %s", err)

		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to create cluster %s %s", res.Status, body)
		return fmt.Errorf("unable to create cluster %s %s", res.Status, body)
	}

	log.Info().Msgf("Created cluster: %s", string(body))

	return nil
}

func ResetClusterProgress(clusterName string) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyResetClusterRequest{
		Url: fmt.Sprintf("/cluster/%s/reset_progress", clusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", GetConsoleIngresUrl()), bytes.NewReader(payload))
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to create cluster %s", res.Status)
		return err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Info().Msgf("unable to create cluster %s", err)

		return err
	}

	log.Info().Msgf("Import: %s", string(body))

	return nil
}

func GetCluster(clusterName string) (apiTypes.Cluster, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	cluster := apiTypes.Cluster{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/proxy?url=/cluster/%s", GetConsoleIngresUrl(), clusterName), nil)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return cluster, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return cluster, err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to get cluster %s, continuing", res.Status)
		return cluster, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Info().Msgf("unable to get cluster %s", err)
		return cluster, err
	}

	err = json.Unmarshal(body, &cluster)
	if err != nil {
		log.Info().Msgf("unable to cast cluster object %s", err)
		return cluster, err
	}

	return cluster, nil
}

func GetClusters() ([]apiTypes.Cluster, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	clusters := []apiTypes.Cluster{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/proxy?url=/cluster", GetConsoleIngresUrl()), nil)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return clusters, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return clusters, err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to get clusters %s, continuing", res.Status)
		return clusters, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Info().Msgf("unable to get clusters %s", err)
		return clusters, err
	}

	err = json.Unmarshal(body, &clusters)
	if err != nil {
		log.Info().Msgf("unable to cast clusters object %s", err)
		return clusters, err
	}

	return clusters, nil
}

func DeleteCluster(clusterName string) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/proxy?url=/cluster/%s", GetConsoleIngresUrl(), clusterName), nil)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Info().Msgf("error %s", err)
		return err
	}

	if res.StatusCode != http.StatusOK {
		log.Info().Msgf("unable to delete cluster %s, continuing", res.Status)
		return err
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		log.Info().Msgf("unable to delete cluster %s", err)
		return err
	}

	return nil
}
