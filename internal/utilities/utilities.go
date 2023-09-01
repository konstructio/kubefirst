/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kubefirst/runtime/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
	// viper.Set("flags.dns-provider", dnsProviderFlag)
	// viper.Set("flags.git-protocol", gitProtocolFlag)

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
