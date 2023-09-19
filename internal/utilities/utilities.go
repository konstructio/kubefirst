/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kubefirst/runtime/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Uncomment this for debbuging purposes
// var ConsoleIngresUrl = "http://localhost:3000"
var ConsoleIngresUrl = "https://console.kubefirst.dev"

// CreateK1ClusterDirectory
func CreateK1ClusterDirectory(clusterName string) {
	// Create k1 dir if it doesn't exist
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Info().Msg(err.Error())
	}
	k1Dir := fmt.Sprintf("%s/.k1/%s", homePath, clusterName)
	if _, err := os.Stat(k1Dir); os.IsNotExist(err) {
		err := os.MkdirAll(k1Dir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", k1Dir)
		}
	}
}

const (
	exportFilePath = "/tmp/api/cluster/export"
)

func CreateClusterRecordFromRaw(useTelemetry bool, gitOwner string, gitUser string, gitToken string, gitlabOwnerGroupID int, gitopsTemplateURL string, gitopsTemplateBranch string) types.Cluster {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	cl := types.Cluster{
		ID:                    primitive.NewObjectID(),
		CreationTimestamp:     fmt.Sprintf("%v", time.Now().UTC()),
		UseTelemetry:          useTelemetry,
		Status:                "provisioned",
		AlertsEmail:           viper.GetString("flags.alerts-email"),
		ClusterName:           viper.GetString("flags.cluster-name"),
		CloudProvider:         cloudProvider,
		CloudRegion:           viper.GetString("flags.cloud-region"),
		DomainName:            domainName,
		ClusterID:             viper.GetString("kubefirst.cluster-id"),
		ClusterType:           "mgmt",
		GitopsTemplateURL:     gitopsTemplateURL,
		GitopsTemplateBranch:  gitopsTemplateBranch,
		GitProvider:           gitProvider,
		GitHost:               fmt.Sprintf("%s.com", gitProvider),
		GitProtocol:           viper.GetString("flags.git-protocol"),
		DnsProvider:           viper.GetString("flags.dns-provider"),
		GitlabOwnerGroupID:    gitlabOwnerGroupID,
		AtlantisWebhookSecret: viper.GetString("secrets.atlantis-webhook"),
		AtlantisWebhookURL:    fmt.Sprintf("https://atlantis.%s/events", domainName),
		KubefirstTeam:         kubefirstTeam,
		ArgoCDAuthToken:       viper.GetString("components.argocd.auth-token"),
		ArgoCDPassword:        viper.GetString("components.argocd.password"),
		GitAuth: types.GitAuth{
			Token:      gitToken,
			User:       gitUser,
			Owner:      gitOwner,
			PublicKey:  viper.GetString("kbot.public-key"),
			PrivateKey: viper.GetString("kbot.private-key"),
		},
		CloudflareAuth: types.CloudflareAuth{
			Token: os.Getenv("CF_API_TOKEN"),
		},
	}

	switch cloudProvider {
	case "civo":
		cl.CivoAuth.Token = os.Getenv("CIVO_TOKEN")
	case "aws":
		//ToDo: where to get credentials?
		cl.AWSAuth.AccessKeyID = viper.GetString("kubefirst.state-store-creds.access-key-id")
		cl.AWSAuth.SecretAccessKey = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
		cl.AWSAuth.SessionToken = viper.GetString("kubefirst.state-store-creds.token")
	case "digitalocean":
		cl.DigitaloceanAuth.Token = os.Getenv("DO_TOKEN")
		cl.DigitaloceanAuth.SpacesKey = os.Getenv("DO_SPACES_KEY")
		cl.DigitaloceanAuth.SpacesSecret = os.Getenv("DO_SPACES_SECRET")
	case "vultr":
		cl.VultrAuth.Token = os.Getenv("VULTR_API_KEY")
	}

	cl.StateStoreCredentials.AccessKeyID = viper.GetString("kubefirst.state-store-creds.access-key-id")
	cl.StateStoreCredentials.SecretAccessKey = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
	cl.StateStoreCredentials.SessionToken = viper.GetString("kubefirst.state-store-creds.token")
	cl.StateStoreCredentials.Name = viper.GetString("kubefirst.state-store-creds.name")
	cl.StateStoreCredentials.ID = viper.GetString("kubefirst.state-store-creds.id")

	cl.StateStoreDetails.ID = viper.GetString("kubefirst.state-store.id")
	cl.StateStoreDetails.Name = viper.GetString("kubefirst.state-store.name")
	cl.StateStoreDetails.Hostname = viper.GetString("kubefirst.state-store.hostname")
	cl.StateStoreDetails.AWSArtifactsBucket = viper.GetString("kubefirst.artifacts-bucket")
	cl.StateStoreDetails.AWSStateStoreBucket = viper.GetString("kubefirst.state-store-bucket")

	return cl
}

func CreateClusterDefinitionRecordFromRaw(gitAuth types.GitAuth, gitopsTemplateURL string, gitopsTemplateBranch string) types.ClusterDefinition {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	cl := types.ClusterDefinition{
		AdminEmail:           viper.GetString("flags.alerts-email"),
		ClusterName:          viper.GetString("flags.cluster-name"),
		CloudProvider:        cloudProvider,
		CloudRegion:          viper.GetString("flags.cloud-region"),
		DomainName:           domainName,
		Type:                 "mgmt",
		GitopsTemplateURL:    gitopsTemplateURL,
		GitopsTemplateBranch: gitopsTemplateBranch,
		GitProvider:          gitProvider,
		GitProtocol:          viper.GetString("flags.git-protocol"),
		DnsProvider:          viper.GetString("flags.dns-provider"),
		GitAuth: types.GitAuth{
			Token:      gitAuth.Token,
			User:       gitAuth.User,
			Owner:      gitAuth.Owner,
			PublicKey:  viper.GetString("kbot.public-key"),
			PrivateKey: viper.GetString("kbot.private-key"),
		},
		CloudflareAuth: types.CloudflareAuth{
			Token: os.Getenv("CF_API_TOKEN"),
		},
	}

	switch cloudProvider {
	case "civo":
		cl.CivoAuth.Token = os.Getenv("CIVO_TOKEN")
	case "aws":
		//ToDo: where to get credentials?
		cl.AWSAuth.AccessKeyID = viper.GetString("kubefirst.state-store-creds.access-key-id")
		cl.AWSAuth.SecretAccessKey = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
		cl.AWSAuth.SessionToken = viper.GetString("kubefirst.state-store-creds.token")
	case "digitalocean":
		cl.DigitaloceanAuth.Token = os.Getenv("DO_TOKEN")
		cl.DigitaloceanAuth.SpacesKey = os.Getenv("DO_SPACES_KEY")
		cl.DigitaloceanAuth.SpacesSecret = os.Getenv("DO_SPACES_SECRET")
	case "vultr":
		cl.VultrAuth.Token = os.Getenv("VULTR_API_KEY")
	}

	return cl
}

func CreateClusterRecordFile(clustername string, cluster types.Cluster) error {
	var localFilePath = fmt.Sprintf("%s/%s.json", exportFilePath, clustername)

	log.Info().Msgf("creating export file %s", localFilePath)

	if _, err := os.Stat(exportFilePath); os.IsNotExist(err) {
		log.Info().Msgf("cluster exports directory does not exist, creating")
		err := os.MkdirAll(exportFilePath, 0777)
		if err != nil {
			return err
		}
	}

	file, _ := json.MarshalIndent(cluster, "", " ")
	_ = os.WriteFile(localFilePath, file, 0644)

	log.Info().Msgf("file created %s", localFilePath)

	return nil
}

func CreateCluster(cluster types.ClusterDefinition) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyCreateClusterRequest{
		Body: cluster,
		Url:  fmt.Sprintf("/cluster/%s", cluster.ClusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", ConsoleIngresUrl), bytes.NewReader(payload))
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

	log.Info().Msgf("Created cluster: %s", string(body))

	return nil
}

func ResetClusterProgress(clusterName string) error {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	requestObject := types.ProxyResetClusterRequest{
		Url: fmt.Sprintf("/cluster/%s/reset_progress", clusterName),
	}

	payload, err := json.Marshal(requestObject)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/proxy", ConsoleIngresUrl), bytes.NewReader(payload))
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

func GetCluster(clusterName string) (types.Cluster, error) {
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	httpClient := http.Client{Transport: customTransport}

	cluster := types.Cluster{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/proxy?url=/cluster/%s", ConsoleIngresUrl, clusterName), nil)
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
		log.Info().Msgf("unable to get cluster %s", res.Status)
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
