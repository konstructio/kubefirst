/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package civo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitShim"
	"github.com/kubefirst/kubefirst/internal/launch"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/kubefirst/kubefirst/internal/utilities"
	"github.com/kubefirst/runtime/pkg"
	internalssh "github.com/kubefirst/runtime/pkg/ssh"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createCivo(cmd *cobra.Command, args []string) error {
	progress.DisplayLogHints()

	alertsEmailFlag, err := cmd.Flags().GetString("alerts-email")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	// ToDo: do we still need this one?
	// ciFlag, err := cmd.Flags().GetBool("ci")
	// if err != nil {
	// 	progress.Error(err.Error())
	// 	return err
	// }

	cloudRegionFlag, err := cmd.Flags().GetString("cloud-region")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	clusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	dnsProviderFlag, err := cmd.Flags().GetString("dns-provider")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	domainNameFlag, err := cmd.Flags().GetString("domain-name")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	githubOrgFlag, err := cmd.Flags().GetString("github-org")
	if err != nil {
		progress.Error(err.Error())
		return err
	}
	githubOrgFlag = strings.ToLower(githubOrgFlag)

	gitlabGroupFlag, err := cmd.Flags().GetString("gitlab-group")
	if err != nil {
		progress.Error(err.Error())
		return err
	}
	gitlabGroupFlag = strings.ToLower(gitlabGroupFlag)

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	gitProtocolFlag, err := cmd.Flags().GetString("git-protocol")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	gitopsTemplateURLFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	// useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	// if err != nil {
	// 	progress.Error(err.Error())
	// 	return err
	// }

	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		return fmt.Errorf("this cluster install process has already completed successfully")
	}

	progress.Log(":female_detective: Validating provided flags", "")

	err = ValidateProvidedFlags(gitProviderFlag)

	if err != nil {
		progress.Error(err.Error())
		return err
	}

	utilities.CreateK1ClusterDirectory(clusterNameFlag)

	// required for destroy command
	viper.Set("flags.alerts-email", alertsEmailFlag)
	viper.Set("flags.cluster-name", clusterNameFlag)
	viper.Set("flags.dns-provider", dnsProviderFlag)
	viper.Set("flags.domain-name", domainNameFlag)
	viper.Set("flags.git-provider", gitProviderFlag)
	viper.Set("flags.git-protocol", gitProtocolFlag)
	viper.Set("flags.cloud-region", cloudRegionFlag)
	viper.Set("kubefirst.cloud-provider", "civo")

	viper.WriteConfig()

	progress.Log(":male_detective: Validating git credentials", "")

	gitAuth, err := gitShim.ValidateGitCredentials(gitProviderFlag, githubOrgFlag, gitlabGroupFlag)

	if err != nil {
		progress.Error("unable to validate git credentials")
	}

	// Validate git
	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", gitProviderFlag))
	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  gitProviderFlag,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		progress.Log(":dizzy: Validating git environment", "")
		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			progress.Error(err.Error())
		}
	}
	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", gitProviderFlag), true)
	viper.WriteConfig()

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	if !k3dClusterCreationComplete {
		launch.Up(nil, true, useTelemetryFlag)
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", "https://console.kubefirst.dev"), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
	}

	cluster := utilities.CreateClusterDefinitionRecordFromRaw(
		gitAuth,
		gitopsTemplateURLFlag,
		gitopsTemplateBranchFlag,
	)

	if cluster.GitopsTemplateBranch == "" {
		cluster.GitopsTemplateBranch = configs.K1Version

		if configs.K1Version == "development" {
			cluster.GitopsTemplateBranch = "main"
		}
	}

	clusterCreated, err := utilities.GetCluster(cluster.ClusterName)
	if err != nil {
		log.Info().Msg("cluster not found")
	}

	if !clusterCreated.InProgress {
		err := utilities.CreateCluster(cluster)
		if err != nil {
			progress.Error("Unable to create the cluster")
		}
	}

	if clusterCreated.Status == "error" {
		utilities.ResetClusterProgress(cluster.ClusterName)
		utilities.CreateCluster(cluster)
	}

	time.Sleep(time.Second * 2)
	progress.StartProvisioning(cluster.ClusterName, 10)

	return nil
}

func ValidateProvidedFlags(gitProviderFlag string) error {
	if os.Getenv("CIVO_TOKEN") == "" {
		// telemetryShim.Transmit(useTelemetryFlag, segmentClient, segment.MetricCloudCredentialsCheckFailed, "CIVO_TOKEN environment variable was not set")
		return fmt.Errorf("your CIVO_TOKEN is not set - please set and re-run your last command")
	}

	if os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("your GITHUB_TOKEN is not set. Please set and try again")
	}
	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProviderFlag {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "github.com", key.Type())
		}
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy")
		} else {
			log.Info().Msgf("%s %s\n", "gitlab.com", key.Type())
		}
	}

	return nil
}
