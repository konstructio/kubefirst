/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package utilities

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/konstructio/kubefirst-api/pkg/configs"
	"github.com/konstructio/kubefirst-api/pkg/k8s"
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/types"
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

func CreateClusterRecordFromRaw(
	useTelemetry bool,
	gitOwner string,
	gitUser string,
	gitToken string,
	gitlabOwnerGroupID int,
	gitopsTemplateURL string,
	gitopsTemplateBranch string,
	catalogApps []apiTypes.GitopsCatalogApp,
) apiTypes.Cluster {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	cl := apiTypes.Cluster{
		ID:                     primitive.NewObjectID(),
		CreationTimestamp:      fmt.Sprintf("%v", time.Now().UTC()),
		UseTelemetry:           useTelemetry,
		Status:                 "provisioned",
		AlertsEmail:            viper.GetString("flags.alerts-email"),
		ClusterName:            viper.GetString("flags.cluster-name"),
		CloudProvider:          cloudProvider,
		CloudRegion:            viper.GetString("flags.cloud-region"),
		DomainName:             domainName,
		ClusterID:              viper.GetString("kubefirst.cluster-id"),
		ClusterType:            "mgmt",
		GitopsTemplateURL:      gitopsTemplateURL,
		GitopsTemplateBranch:   gitopsTemplateBranch,
		GitProvider:            gitProvider,
		GitHost:                fmt.Sprintf("%s.com", gitProvider),
		GitProtocol:            viper.GetString("flags.git-protocol"),
		DnsProvider:            viper.GetString("flags.dns-provider"),
		GitlabOwnerGroupID:     gitlabOwnerGroupID,
		AtlantisWebhookSecret:  viper.GetString("secrets.atlantis-webhook"),
		AtlantisWebhookURL:     fmt.Sprintf("https://atlantis.%s/events", domainName),
		KubefirstTeam:          kubefirstTeam,
		ArgoCDAuthToken:        viper.GetString("components.argocd.auth-token"),
		ArgoCDPassword:         viper.GetString("components.argocd.password"),
		PostInstallCatalogApps: catalogApps,
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
		// ToDo: where to get credentials?
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

func CreateClusterDefinitionRecordFromRaw(gitAuth apiTypes.GitAuth, cliFlags types.CliFlags, catalogApps []apiTypes.GitopsCatalogApp) apiTypes.ClusterDefinition {
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	domainName := viper.GetString("flags.domain-name")
	gitProvider := viper.GetString("flags.git-provider")

	kubefirstTeam := os.Getenv("KUBEFIRST_TEAM")
	if kubefirstTeam == "" {
		kubefirstTeam = "false"
	}

	stringToIntNodeCount, err := strconv.Atoi(cliFlags.NodeCount)
	if err != nil {
		log.Info().Msg("Unable to convert node count to type string")
	}

	cl := apiTypes.ClusterDefinition{
		AdminEmail:             viper.GetString("flags.alerts-email"),
		ClusterName:            viper.GetString("flags.cluster-name"),
		CloudProvider:          cloudProvider,
		CloudRegion:            viper.GetString("flags.cloud-region"),
		DomainName:             domainName,
		SubdomainName:          cliFlags.SubDomainName,
		Type:                   "mgmt",
		NodeType:               cliFlags.NodeType,
		NodeCount:              stringToIntNodeCount,
		GitopsTemplateURL:      cliFlags.GitopsTemplateURL,
		GitopsTemplateBranch:   cliFlags.GitopsTemplateBranch,
		GitopsRepoName:         cliFlags.GitopsRepoName,
		MetaphorRepoName:       cliFlags.MetaphorRepoName,
		AdminTeamName:          cliFlags.AdminTeamName,
		DeveloperTeamName:      cliFlags.DeveloperTeamName,
		GitProvider:            gitProvider,
		GitProtocol:            viper.GetString("flags.git-protocol"),
		DnsProvider:            viper.GetString("flags.dns-provider"),
		LogFileName:            viper.GetString("k1-paths.log-file-name"),
		PostInstallCatalogApps: catalogApps,
		InstallKubefirstPro:    cliFlags.InstallKubefirstPro,
		GitAuth: apiTypes.GitAuth{
			Token:      gitAuth.Token,
			User:       gitAuth.User,
			Owner:      gitAuth.Owner,
			PublicKey:  viper.GetString("kbot.public-key"),
			PrivateKey: viper.GetString("kbot.private-key"),
		},
		CloudflareAuth: apiTypes.CloudflareAuth{
			APIToken: os.Getenv("CF_API_TOKEN"),
		},
	}

	if cl.GitopsTemplateBranch == "" {
		cl.GitopsTemplateBranch = configs.K1Version

		if configs.K1Version == "development" {
			cl.GitopsTemplateBranch = "main"
		}
	}

	switch cloudProvider {
	case "akamai":
		cl.AkamaiAuth.Token = os.Getenv("LINODE_TOKEN")
	case "aws":
		// ToDo: where to get credentials?
		cl.AWSAuth.AccessKeyID = viper.GetString("kubefirst.state-store-creds.access-key-id")
		cl.AWSAuth.SecretAccessKey = viper.GetString("kubefirst.state-store-creds.secret-access-key-id")
		cl.AWSAuth.SessionToken = viper.GetString("kubefirst.state-store-creds.token")
		cl.ECR = cliFlags.Ecr
	case "civo":
		cl.CivoAuth.Token = os.Getenv("CIVO_TOKEN")
	case "digitalocean":
		cl.DigitaloceanAuth.Token = os.Getenv("DO_TOKEN")
		cl.DigitaloceanAuth.SpacesKey = os.Getenv("DO_SPACES_KEY")
		cl.DigitaloceanAuth.SpacesSecret = os.Getenv("DO_SPACES_SECRET")
	case "vultr":
		cl.VultrAuth.Token = os.Getenv("VULTR_API_KEY")
	case "k3s":
		cl.K3sAuth.K3sServersPrivateIps = viper.GetStringSlice("flags.servers-private-ips")
		cl.K3sAuth.K3sServersPublicIps = viper.GetStringSlice("flags.servers-public-ips")
		cl.K3sAuth.K3sSshUser = viper.GetString("flags.ssh-user")
		cl.K3sAuth.K3sSshPrivateKey = viper.GetString("flags.ssh-privatekey")
		cl.K3sAuth.K3sServersArgs = viper.GetStringSlice("flags.servers-args")
	case "google":
		jsonFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

		jsonFile, err := os.Open(jsonFilePath)
		if err != nil {
			progress.Error("Unable to read GOOGLE_APPLICATION_CREDENTIALS file")
		}
		defer jsonFile.Close()

		jsonContent, _ := ioutil.ReadAll(jsonFile)

		cl.GoogleAuth.KeyFile = string(jsonContent)
		cl.GoogleAuth.ProjectId = cliFlags.GoogleProject
	}

	return cl
}

func ExportCluster(cluster apiTypes.Cluster, kcfg *k8s.KubernetesClient) error {
	cluster.Status = "provisioned"
	cluster.InProgress = false

	if viper.GetBool("kubefirst-checks.secret-export-state") {
		return nil
	}

	time.Sleep(time.Second * 10)

	bytes, err := json.Marshal(cluster)
	if err != nil {
		log.Error().Msg(err.Error())
		return err
	}

	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	secret := &v1secret.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "kubefirst-initial-state", Namespace: "kubefirst"},
		Data:       secretValuesMap,
	}

	err = k8s.CreateSecretV2(kcfg.Clientset, secret)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("unable to save secret to management cluster. %s", err))
	}

	viper.Set("kubefirst-checks.secret-export-state", true)
	viper.WriteConfig()

	return nil
}

func ParseJSONToMap(jsonStr string) (map[string][]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	for key, value := range result {
		switch v := value.(type) {
		case map[string]interface{}, []interface{}: // For nested structures, marshal back to JSON
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			secretData[key] = bytes
		}
	}

	return secretData, nil
}
