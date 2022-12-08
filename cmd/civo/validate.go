package civo

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/cip8/autoname"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/handlers"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// validateCivo is responsible for gathering all of the information required to execute a kubefirst civo cloud creation with github (currently)
// this function needs to provide all the generated values and provides a single space for writing and updating configuration up front.
func validateCivo(cmd *cobra.Command, args []string) error {

	// todo emit init telemetry begin

	config := configs.ReadConfig()

	//* get cli flag values for storage in `$HOME/.kubefirst`
	adminEmailFlag, err := cmd.Flags().GetString("admin-email")
	if err != nil {
		log.Panic(err)
	}

	cloudProviderFlag, err := cmd.Flags().GetString("cloud-provider")
	if err != nil {
		log.Panic(err)
	}

	civoDnsFlag, err := cmd.Flags().GetString("dns")
	if err != nil {
		log.Panic(err)
	}

	civoClusterNameFlag, err := cmd.Flags().GetString("cluster-name")
	if err != nil {
		log.Panic(err)
	}
	log.Println(civoClusterNameFlag)

	githubOwnerFlag, err := cmd.Flags().GetString("github-owner")
	if err != nil {
		log.Panic(err)
	}

	gitopsTemplateUrlFlag, err := cmd.Flags().GetString("gitops-template-url")
	if err != nil {
		log.Panic(err)
	}

	gitopsTemplateBranchFlag, err := cmd.Flags().GetString("gitops-template-branch")
	if err != nil {
		log.Panic(err)
	}
	log.Println(gitopsTemplateBranchFlag)

	gitProviderFlag, err := cmd.Flags().GetString("git-provider")
	if err != nil {
		log.Panic(err)
	}
	kbotPasswordFlag, err := cmd.Flags().GetString("kbot-password")
	if err != nil {
		log.Panic(err)
	}

	silentModeFlag, err := cmd.Flags().GetBool("silent-mode")
	if err != nil {
		log.Panic(err)
	}
	log.Println(silentModeFlag)

	useTelemetryFlag, err := cmd.Flags().GetBool("use-telemetry")
	if err != nil {
		log.Panic(err)
	}
	log.Println(useTelemetryFlag)

	//! hack
	// if err := pkg.ValidateK1Folder(config.K1FolderPath); err != nil {
	// 	return err
	// }

	// todo validate flags
	viper.Set("admin-email", adminEmailFlag)
	viper.Set("argocd.local.service", config.ArgocdLocalUrl)
	viper.Set("cloud-provider", cloudProviderFlag)
	viper.Set("git-provider", gitProviderFlag)
	viper.Set("template-repo.gitops.branch", gitopsTemplateBranchFlag)
	viper.Set("template-repo.gitops.url", gitopsTemplateUrlFlag)
	// todo accommodate metaphor branch and repo override more intelligently
	viper.Set("template-repo.metaphor.url", fmt.Sprintf("https://github.com/%s/metaphor.git", "kubefirst"))
	viper.Set("template-repo.metaphor.branch", "main")
	viper.Set("template-repo.metaphor-frontend.url", fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", "kubefirst"))
	viper.Set("template-repo.metaphor-frontend.branch", "main")
	viper.Set("template-repo.metaphor-go.url", fmt.Sprintf("https://github.com/%s/metaphor-go.git", "kubefirst"))
	viper.Set("template-repo.metaphor-go.branch", "main")
	viper.Set("github.atlantis.webhook.secret", pkg.Random(20))
	viper.Set("github.atlantis.webhook.url", fmt.Sprintf("https://atlantis.%s/events", civoDnsFlag))
	viper.Set("github.repo.gitops.url", fmt.Sprintf("https://github.com/%s/gitops.git", githubOwnerFlag))
	viper.Set("github.repo.metaphor.url", fmt.Sprintf("https://github.com/%s/metaphor.git", githubOwnerFlag))
	viper.Set("github.repo.metaphor-frontend.url", fmt.Sprintf("https://github.com/%s/metaphor-frontend.git", githubOwnerFlag))
	viper.Set("github.repo.metaphor-go.url", fmt.Sprintf("https://github.com/%s/metaphor-go.git", githubOwnerFlag))
	githubOwnerRootGitUrl := fmt.Sprintf("git@github.com:%s", githubOwnerFlag)
	viper.Set("github.repo.gitops.giturl", fmt.Sprintf("%s/gitops.git", githubOwnerRootGitUrl))
	viper.WriteConfig()

	pkg.InformUser("checking authentication to required providers", silentModeFlag)
	//* check CIVO_TOKEN environment variable
	civoToken := os.Getenv("CIVO_TOKEN")
	if civoToken == "" {
		fmt.Println("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security to retrieve your token\nand enter it here, then press Enter:")
		var civoToken string
		fmt.Scanln(&civoToken)
		log.Printf(civoToken)
		os.Setenv("CIVO_TOKEN", civoToken)
		log.Printf("CIVO TOKEN %s", os.Getenv("CIVO_TOKEN"))
	}
	log.Println("CIVO_TOKEN is set")

	//* github checks
	executionControl := viper.GetBool("kubefirst.checks.github.complete")
	if !executionControl {
		httpClient := http.DefaultClient
		githubToken := config.GithubToken
		if len(githubToken) == 0 {
			return errors.New("ephemeral tokens not supported for cloud installations, please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
		}
		gitHubService := services.NewGitHubService(httpClient)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		// todo ask for user input here
		// todo 1. enter github personal access token
		// todo 2. generate temporary token with device login
		// todo write temporary token to viper ignore env
		// todo github.repo.*.url check if these repos exist
		// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
		// todo look up teams

		// get Github data to set user based on the provided token
		log.Println("verifying github user")
		githubUser, err := gitHubHandler.GetGitHubUser(githubToken)
		if err != nil {
			return err
		}
		log.Println("github user is: ", githubUser)
		// todo evaluate if cloudProviderFlag == "local" {githubOwnerFlag = githubUser} and the rest of the execution is the same

		err = gitHubHandler.CheckGithubOrganizationPermissions(githubToken, githubOwnerFlag, githubUser)
		if err != nil {
			return err
		}

		githubWrapper := githubWrapper.New()

		newRepositoryExists := false
		newRepositoryNames := []string{"gitops", "metaphor", "metaphor-frontend", "metaphor-go"}
		errorMsg := "the following repositories must be removed before continuing with your kubefirst installation.\n\t"

		for _, repositoryName := range newRepositoryNames {
			responseStatusCode := githubWrapper.CheckRepoExists(githubOwnerFlag, repositoryName)

			// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#get-a-repository
			repositoryExistsStatusCode := 200
			repositoryDoesNotExistStatusCode := 404

			if responseStatusCode == repositoryExistsStatusCode {
				log.Printf("repository https://github.com/%s/%s exists", githubOwnerFlag, repositoryName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/%s/%s\n\t", githubOwnerFlag, repositoryName)
				newRepositoryExists = true
			} else if responseStatusCode == repositoryDoesNotExistStatusCode {
				log.Printf("repository https://github.com/%s/%s does not exist, continuing", githubOwnerFlag, repositoryName)
			}
		}
		if newRepositoryExists {
			return errors.New(errorMsg)
		}

		newTeamExists := false
		newTeamNames := []string{"admins", "developers"}
		errorMsg = "the following teams must be removed before continuing with your kubefirst installation.\n\t"

		for _, teamName := range newTeamNames {
			responseStatusCode := githubWrapper.CheckTeamExists(githubOwnerFlag, teamName)

			// https://docs.github.com/en/rest/teams/teams?apiVersion=2022-11-28#get-a-team-by-name
			teamExistsStatusCode := 200
			teamDoesNotExistStatusCode := 404

			if responseStatusCode == teamExistsStatusCode {
				log.Printf("team https://github.com/%s/%s exists", githubOwnerFlag, teamName)
				errorMsg = errorMsg + fmt.Sprintf("https://github.com/orgs/%s/teams/%s\n\t", githubOwnerFlag, teamName)
				newTeamExists = true
			} else if responseStatusCode == teamDoesNotExistStatusCode {
				log.Printf("https://github.com/orgs/%s/teams/%s does not exist, continuing", githubOwnerFlag, teamName)
			}
		}
		if newTeamExists {
			return errors.New(errorMsg)
		}

		viper.Set("github.owner", githubOwnerFlag)
		viper.Set("github.user", githubUser)
		viper.Set("kubefirst.checks.github.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed github checks - continuing")
	}

	executionControl = viper.GetBool("kubefirst.checks.civo.complete")
	if !executionControl {
		log.Println("getting civo account information")
		// civoAccountId, civoIamArn, civoRegion, err := civo.GetAccountInfoV2(civoProfileFlag, civoRegionFlag)
		// if err != nil {
		// 	return err
		// }
		// log.Printf("civo account id: %s\ncivo user arn: %s", civoAccountId, civoIamArn)

		// log.Println("getting civo hosted zone id for zone ", civoDnsFlag)
		// civoHostedZoneId := civo.GetHostedZoneId(civoProfileFlag, civoRegion, civoHostedZoneNameFlag)
		// log.Printf("civo hosted zone id %s", civoHostedZoneId)

		log.Printf("creating state store bucket ")
		randomName := strings.ReplaceAll(autoname.Generate(), "_", "-")

		kubefirstStateStoreBucketName := fmt.Sprintf("k1-state-store-%s", randomName)
		// todo consider creating a bucket in civo cloud just like aws
		// err = civo.CreateS3Bucket(civoProfileFlag, civoRegionFlag, civoClusterNameFlag, kubefirstStateStoreBucketName)
		if err != nil {
			log.Printf("creating state store bucket ")
			return err
		}
		viper.Set("kubefirst.state-store.bucket", kubefirstStateStoreBucketName)
		viper.Set("kubefirst.bucket.random-name", randomName)
		viper.Set("kubefirst.telemetry", useTelemetryFlag)
		viper.Set("cluster-name", civoClusterNameFlag)
		viper.Set("vault.local.service", config.VaultLocalUrl)
		viper.Set("civo.dns", civoDnsFlag)
		viper.Set("civo.region", civoRegionFlag)
		viper.Set("kubefirst.checks.civo.complete", true)
		viper.WriteConfig()
	} else {
		log.Println("already completed civo checks - continuing")
	}

	executionControl = viper.GetBool("kubefirst.checks.bot.complete")
	if !executionControl {
		log.Println("creating an ssh key pair for your new cloud infrastructure")
		sshPrivateKey, sshPublicKey, err := pkg.CreateSshKeyPair()
		if err != nil {
			return err
		}
		if len(kbotPasswordFlag) == 0 {
			kbotPasswordFlag = pkg.Random(20)
		}
		log.Println("ssh key pair creation complete")
		viper.Set("kubefirst.bot.password", kbotPasswordFlag)
		viper.Set("kubefirst.bot.private-key", sshPrivateKey)
		viper.Set("kubefirst.bot.public-key", sshPublicKey)
		viper.Set("kubefirst.bot.user", "kbot")
		viper.Set("kubefirst.checks.bot.complete", true)
		viper.WriteConfig()
		// todo, is this a hangover from initial gitlab? do we need this?
		log.Println("creating argocd-init-values.yaml for initial install")
		//* ex: `git@github.com:kubefirst` this is allows argocd access to the kubefirst organization repos
		err = pkg.WriteGithubArgoCdInitValuesFile(githubOwnerRootGitUrl, sshPrivateKey)
		if err != nil {
			return err
		}
		log.Println("argocd-init-values.yaml creation complete")
	}

	log.Println("validation and kubefirst cli environment check is complete")

	// if useTelemetryFlag {
	// 	if err := wrappers.SendSegmentIoTelemetry(civoHostedZoneNameFlag, pkg.MetricInitCompleted); err != nil {
	// 		log.Println(err)
	// 	}
	// }

	// todo progress bars
	// time.Sleep(time.Millisecond * 100) // to allow progress bars to finish

	return nil
}
