/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsinternal "github.com/konstructio/kubefirst-api/pkg/aws"
	"github.com/konstructio/kubefirst-api/pkg/gitClient"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	"github.com/konstructio/kubefirst-api/pkg/terraform"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createAws(cmd *cobra.Command, _ []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "aws")
	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	progress.DisplayLogHints(40)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		return fmt.Errorf("invalid catalog apps: %w", err)
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	utilities.CreateK1ClusterDirectory(cliFlags.ClusterName)

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err = fmt.Errorf("this cluster install process has already completed successfully")
		progress.Error(err.Error())
		return nil
	}

	// Validate aws region
	config, err := awsinternal.NewAwsV2(cloudRegionFlag)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate AWS region: %w", err)
	}

	awsClient := &awsinternal.Configuration{Config: config}
	creds, err := awsClient.Config.Credentials.Retrieve(aws.BackgroundContext())
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
	viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	_, err = awsClient.CheckAvailabilityZones(cliFlags.CloudRegion)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to check availability zones: %w", err)
	}

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate Git credentials: %w", err)
	}

	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider))
	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  cliFlags.GitProvider,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
		return fmt.Errorf("failed to check kubefirst API availability: %w", err)
	}

	if err := provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps); err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to create management cluster: %w", err)
	}

	return nil
}

func ValidateProvidedFlags(gitProvider string) error {
	progress.AddStep("Validate provided flags")

	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "github.com", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "gitlab.com", key.Type())
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}

func ConnectAWS(cmd *cobra.Command, _ []string) error {
	flags, err := utilities.GetConnectFlags(cmd, "aws")
	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error().Msgf("something went wrong getting home path: %s", err)
		return fmt.Errorf("unable to get home path: %w", err)
	}
	clusterName := viper.GetString("flags.cluster-name")
	terraformClient := fmt.Sprintf("%s/.k1/%s/tools/terraform", homeDir, clusterName)
	arnDir := fmt.Sprintf("%s/.k1/%s/aws-arn", homeDir, clusterName)
	arnRepoURL := "https://github.com/jokestax/aws-arn"
	if _, err := os.Stat(arnDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error creating aws-arn directory : %w", err)
		}

		_, err = gitClient.Clone("main", arnDir, arnRepoURL)
		if err != nil {
			return fmt.Errorf("error cloning repository : %w", err)
		}
	}

	tfEnvs := map[string]string{
		"AWS_ACCESS_KEY_ID":     flags.AWS_ACCESS_KEY_ID,
		"AWS_SECRET_ACCESS_KEY": flags.AWS_SECRET_ACCESS_KEY,
		"TF_VAR_oidc_endpoint":  flags.OIDC_ENDPOINT,
		"TF_VAR_cluster_name":   clusterName,
	}

	tfstateFile := filepath.Join(arnDir, "terraform.tfstate")

	if _, err := os.Stat(tfstateFile); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error reading tfstate file: %w", err)
		}
	} else {
		err = terraform.InitDestroyAutoApprove(terraformClient, arnDir, tfEnvs)
		if err != nil {
			msg := fmt.Errorf("error destroying policy resources with terraform in directory %q: %w", arnDir, err)
			return msg
		}
	}

	err = terraform.InitApplyAutoApprove(terraformClient, arnDir, tfEnvs)
	if err != nil {
		msg := fmt.Errorf("error creating policy resources with terraform %q: %w", arnDir, err)
		return msg
	}
	txtFile := filepath.Join(arnDir, "modules", "kubefirst-pro", "iam_role_arn.txt")
	role_arn, err := os.ReadFile(txtFile)

	if err != nil {
		return fmt.Errorf("error retrieving role arn : %w", err)
	}

	fmt.Println(" \n Role ARN is \n ")
	message := fmt.Sprintf("# %s", role_arn)
	progress.Success(message)
	return nil
}
