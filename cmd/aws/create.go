/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	awsinternal "github.com/konstructio/kubefirst-api/pkg/aws"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func (s *Service) createAwsCluster(cliFlags types.CliFlags) error {
	// TODO - Add progress steps
	// progress.DisplayLogHints(40)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		s.logger.Error("invalid catalog apps", "error", err)
		return fmt.Errorf("invalid catalog apps: %w", err)
	}

	// Validate aws region
	config, err := awsinternal.NewAwsV2(cloudRegionFlag)
	if err != nil {
		s.logger.Error("failed to validate AWS region", "error", err)
		return fmt.Errorf("failed to validate AWS region: %w", err)
	}

	awsClient := &awsinternal.Configuration{Config: config}
	creds, err := awsClient.Config.Credentials.Retrieve(aws.BackgroundContext())
	if err != nil {
		s.logger.Error("failed to retrieve AWS credentials", "error", err)
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
	viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
	if err := viper.WriteConfig(); err != nil {
		s.logger.Error("failed to write config", "error", err)

		return fmt.Errorf("failed to write config: %w", err)
	}

	_, err = awsClient.CheckAvailabilityZones(cliFlags.CloudRegion)
	if err != nil {
		s.logger.Error("failed to check availability zones", "error", err)
		return fmt.Errorf("failed to check availability zones: %w", err)
	}

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
		s.logger.Error("failed to validate Git credentials", "error", err)
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
			s.logger.Error("failed to initialize Git provider", "error", err)
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err := viper.WriteConfig(); err != nil {
		s.logger.Error("failed to write config", "error", err)
		return fmt.Errorf("failed to write config: %w", err)
	}

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		s.logger.Error("failed to check kubefirst API availability", "error", err)
		return fmt.Errorf("failed to check kubefirst API availability: %w", err)
	}

	if err := provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps); err != nil {
		s.logger.Error("failed to create management cluster", "error", err)
		return fmt.Errorf("failed to create management cluster: %w", err)
	}

	return nil
}

func (s *Service) createAws(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(s.writer, "Starting to create AWS cluster")

	cliFlags, err := utilities.GetFlags(cmd, "aws")
	if err != nil {
		s.logger.Error("failed to get flags", "error", err)
		return fmt.Errorf("failed to get flags: %w", err)
	}

	err = ValidateProvidedFlags(cliFlags.GitProvider)
	if err != nil {
		s.logger.Error("failed to validate provided flags", "error", err)
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	// Create k1 cluster directory
	homePath, err := os.UserHomeDir()
	if err != nil {
		s.logger.Error("failed to get user home directory", "error", err)
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	err = utilities.CreateK1ClusterDirectoryE(homePath, cliFlags.ClusterName)
	if err != nil {
		s.logger.Error("failed to create k1 cluster directory", "error", err)
		return fmt.Errorf("failed to create k1 cluster directory: %w", err)
	}

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		s.logger.Info("cluster install process has already completed successfully")
		fmt.Fprintln(s.writer, "Cluster install process has already completed successfully")
		return nil
	}

	err = s.createAwsCluster(cliFlags)
	if err != nil {
		s.logger.Error("failed to create AWS cluster", "error", err)
		return fmt.Errorf("failed to create AWS cluster: %w", err)
	}

	fmt.Fprintln(s.writer, "AWS cluster creation complete")

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
