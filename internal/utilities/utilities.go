/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	apiTypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/kubefirst/internal/types"
	"github.com/kubefirst/runtime/pkg/k8s"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	v1secret "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func CreateClusterRecordFromRaw(useTelemetry bool, gitOwner string, gitUser string, gitToken string, gitlabOwnerGroupID int, gitopsTemplateURL string, gitopsTemplateBranch string) apiTypes.Cluster {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	cl := apiTypes.Cluster{
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
		GitAuth: apiTypes.GitAuth{
			Token:      gitToken,
			User:       gitUser,
			Owner:      gitOwner,
			PublicKey:  viper.GetString("kbot.public-key"),
			PrivateKey: viper.GetString("kbot.private-key"),
		},
		CloudflareAuth: apiTypes.CloudflareAuth{
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

func CreateClusterDefinitionRecordFromRaw(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags) apiTypes.ClusterDefinition {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	cl := apiTypes.ClusterDefinition{
		AdminEmail:           viper.GetString("flags.alerts-email"),
		ClusterName:          viper.GetString("flags.cluster-name"),
		CloudProvider:        cloudProvider,
		CloudRegion:          viper.GetString("flags.cloud-region"),
		DomainName:           domainName,
		Type:                 "mgmt",
		GitopsTemplateURL:    cliFlags.GitopsTemplateURL,
		GitopsTemplateBranch: cliFlags.GitopsTemplateBranch,
		GitProvider:          gitProvider,
		GitProtocol:          viper.GetString("flags.git-protocol"),
		DnsProvider:          viper.GetString("flags.dns-provider"),
		GitAuth: apiTypes.GitAuth{
			Token:      gitAuth.Token,
			User:       gitAuth.User,
			Owner:      gitAuth.Owner,
			PublicKey:  viper.GetString("kbot.public-key"),
			PrivateKey: viper.GetString("kbot.private-key"),
		},
		CloudflareAuth: apiTypes.CloudflareAuth{
			Token: os.Getenv("CF_API_TOKEN"),
		},
	}

	if cl.GitopsTemplateBranch == "" {
		cl.GitopsTemplateBranch = configs.K1Version

		if configs.K1Version == "development" {
			cl.GitopsTemplateBranch = "main"
		}
	}

	switch cloudProvider {
	case "civo":
		cl.CivoAuth.Token = os.Getenv("CIVO_TOKEN")
	case "aws":
		//ToDo: where to get credentials?
		cl.AWSAuth.AccessKeyID = viper.GetString("kubefirst.state-store-creds.access-key-id")
		cl.AWSAuth.SecretAccessKey = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
		cl.AWSAuth.SessionToken = viper.GetString("kubefirst.state-store-creds.token")
		cl.ECR = cliFlags.Ecr
	case "digitalocean":
		cl.DigitaloceanAuth.Token = os.Getenv("DO_TOKEN")
		cl.DigitaloceanAuth.SpacesKey = os.Getenv("DO_SPACES_KEY")
		cl.DigitaloceanAuth.SpacesSecret = os.Getenv("DO_SPACES_SECRET")
	case "vultr":
		cl.VultrAuth.Token = os.Getenv("VULTR_API_KEY")
	case "google":
		jsonFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

		jsonFile, err := os.Open(jsonFilePath)
		if err != nil {
			progress.Error("Unable to read GOOGLE_APPLICATION_CREDENTIALS file")
		}

		jsonContent, _ := ioutil.ReadAll(jsonFile)

		cl.GoogleAuth.KeyFile = string(jsonContent)
		cl.GoogleAuth.ProjectId = cliFlags.GoogleProject
	}

	return cl
}

func ExportCluster(cluster apiTypes.Cluster, kcfg *k8s.KubernetesClient) error {
	cluster.Status = "provisioned"
	cluster.InProgress = false

	time.Sleep(time.Second * 10)

	payload, err := json.Marshal(cluster)
	if err != nil {
		log.Error().Msg(err.Error())
		return err
	}

	secret := &v1secret.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "mongodb-state", Namespace: "kubefirst"},
		Data: map[string][]byte{
			"cluster-0":    []byte(payload),
			"cluster-name": []byte(cluster.ClusterName),
		},
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secret)

	if err != nil {
		return errors.New(fmt.Sprintf("unable to save secret to management cluster. %s", err))
	}

	return nil
}
