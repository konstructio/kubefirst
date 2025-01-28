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

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/types"
)

func GetConsoleIngressURL() string {
	if strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true" { // allow using local console running on port 3000
		return os.Getenv("K1_CONSOLE_REMOTE_URL")
	}

	return "https://console.kubefirst.dev"
}

type ClusterClient struct{}

func (c *ClusterClient) CreateCluster(cluster apiTypes.ClusterDefinition) error {
	return CreateCluster(cluster)
}

func (c *ClusterClient) GetCluster(clusterName string) (apiTypes.Cluster, error) {
	return GetCluster(clusterName)
}

func (c *ClusterClient) ResetClusterProgress(clusterName string) error {
	return ResetClusterProgress(clusterName)
}

func CreateCluster(cluster apiTypes.ClusterDefinition) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyCreateClusterRequest{
		Body: cluster,
		URL:  fmt.Sprintf("/cluster/%s", cluster.ClusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return fmt.Errorf("failed to marshal request object: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", GetConsoleIngressURL()), bytes.NewReader(payload))
	if err != nil {
		log.Printf("error creating request: %s", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error executing request: %s", err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("unable to create cluster: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode != http.StatusAccepted {
		log.Printf("unable to create cluster: %q %q", res.Status, body)
		return fmt.Errorf("unable to create cluster: API returned unexpected status code %q: %s", res.Status, body)
	}

	log.Printf("Created cluster: %q", string(body))

	return nil
}

func ResetClusterProgress(clusterName string) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyResetClusterRequest{
		URL: fmt.Sprintf("/cluster/%s/reset_progress", clusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return fmt.Errorf("failed to marshal request object: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", GetConsoleIngressURL()), bytes.NewReader(payload))
	if err != nil {
		log.Printf("error creating request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error executing request: %v", err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("unable to reset cluster progress: %q", res.Status)
		return fmt.Errorf("unable to reset cluster progress: API returned unexpected status %q", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("unable to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Info().Msgf("Import: %s", string(body))
	return nil
}

var ErrNotFound = fmt.Errorf("cluster not found")

func GetCluster(clusterName string) (apiTypes.Cluster, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	cluster := apiTypes.Cluster{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/proxy?url=/cluster/%s", GetConsoleIngressURL(), clusterName), nil)
	if err != nil {
		log.Printf("error creating request: %v", err)
		return cluster, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error executing request: %v", err)
		return cluster, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusNotFound:
		return cluster, ErrNotFound
	case http.StatusOK:
		// continue with the rest
	default:
		log.Printf("unable to get cluster: %q", res.Status)
		return cluster, fmt.Errorf("unable to get cluster: %q", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("unable to read response body: %v", err)
		return cluster, fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, &cluster)
	if err != nil {
		log.Printf("unable to unmarshal cluster object: %v", err)
		return cluster, fmt.Errorf("failed to unmarshal cluster object: %w", err)
	}

	return cluster, nil
}

func GetClusters() ([]apiTypes.Cluster, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	clusters := []apiTypes.Cluster{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/proxy?url=/cluster", GetConsoleIngressURL()), nil)
	if err != nil {
		log.Printf("error creating request: %v", err)
		return clusters, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error executing request: %v", err)
		return clusters, fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("unable to get clusters: %q", res.Status)
		return clusters, fmt.Errorf("unable to get clusters: API returned unexpected status code %q", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("unable to read response body: %v", err)
		return clusters, fmt.Errorf("failed to read response body: %w", err)
	}

	err = json.Unmarshal(body, &clusters)
	if err != nil {
		log.Printf("unable to unmarshal clusters object: %v", err)
		return clusters, fmt.Errorf("failed to unmarshal clusters object: %w", err)
	}

	return clusters, nil
}

func DeleteCluster(clusterName string) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpClient := http.Client{Transport: customTransport}

	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/proxy?url=/cluster/%s", GetConsoleIngressURL(), clusterName), nil)
	if err != nil {
		log.Printf("error creating request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		log.Printf("error executing request: %v", err)
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Printf("unable to delete cluster: %q, continuing", res.Status)
		return fmt.Errorf("unable to delete cluster: API returned unexpected status code %q", res.Status)
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		log.Printf("unable to read response body: %v", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	return nil
}
