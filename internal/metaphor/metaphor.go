package metaphor

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst/pkg"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// DeployMetaphorGitlab - Deploy metaphor applications on gitlab install
func DeployMetaphorGitlab(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Info().Msgf("[#99] Dry-run mode, DeployMetaphorGitlab skipped.")
		return nil
	}
	log.Info().Msgf("cloning and detokenizing the metaphor-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Info().Msgf("clone and detokenization of metaphor-template repository complete")

	log.Info().Msgf("cloning and detokenizing the metaphor-go-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor-go", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Info().Msgf("clone and detokenization of metaphor-go-template repository complete")

	log.Info().Msgf("cloning and detokenizing the metaphor-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Info().Msgf("clone and detokenization of metaphor-template repository complete")

	if !viper.GetBool("gitlab.metaphor-pushed") {
		log.Info().Msgf("Pushing metaphor repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor")
		viper.Set("gitlab.metaphor-pushed", true)
		viper.WriteConfig()
		log.Info().Msgf("clone and detokenization of metaphor-template repository complete")
	}

	// Go template
	if !viper.GetBool("gitlab.metaphor-go-pushed") {
		log.Info().Msgf("Pushing metaphor-go repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor-go")
		viper.Set("gitlab.metaphor-go-pushed", true)
		viper.WriteConfig()
	}

	// Frontend template
	if !viper.GetBool("gitlab.metaphor-pushed") {
		log.Info().Msgf("Pushing metaphor repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor")
		viper.Set("gitlab.metaphor-pushed", true)
		viper.WriteConfig()
	}
	return nil
}

// DeployMetaphorGithub - Deploy metaphor applications on github install
func DeployMetaphorGithub(globalFlags flagset.GlobalFlags) error {
	if globalFlags.DryRun {
		log.Info().Msgf("[#99] Dry-run mode, DeployMetaphorGithub skipped.")
		return nil
	}
	githubOwner := viper.GetString("github.owner")
	githubHost := viper.GetString("github.host")
	if viper.GetBool("github.metaphor-pushed") {
		log.Info().Msgf("github.metaphor-pushed already executed skipped")
		return nil
	}
	config := configs.ReadConfig()

	tfEntrypoint := config.GitOpsLocalRepoPath + "/terraform/github"
	err := os.Rename(fmt.Sprintf("%s/%s", tfEntrypoint, "metaphor-repos.md"), fmt.Sprintf("%s/%s", tfEntrypoint, "metaphor-repos.tf"))
	if err != nil {
		log.Error().Err(err).Msg("error renaming metaphor-repos.md to metaphor-repos.tf")
	}
	gitClient.PushLocalRepoUpdates(githubHost, githubOwner, "gitops", "github")
	terraform.InitApplyAutoApprove(globalFlags.DryRun, tfEntrypoint, map[string]string{})

	repos := [3]string{"metaphor", "metaphor-go", "metaphor"}
	for _, element := range repos {
		log.Info().Msgf("Processing Repo: %s", element)
		repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), element, viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
		log.Info().Msgf("clone and detokenization of %s-template repository complete", element)
		githubHost := viper.GetString("github.host")

		gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, element, "github")

	}

	viper.Set("github.metaphor-pushed", true)
	viper.WriteConfig()
	return nil
}

// DeployMetaphorGithubLocal Deploy metaphor applications on github install
func DeployMetaphorGithubLocal(dryRun bool, skipMetaphor bool, gitHubOwner string, metaphorBranch string, templateTag string) error {
	var err error
	if dryRun {
		log.Info().Msgf("[#99] Dry-run mode, DeployMetaphorGithub skipped.")
		return nil
	}

	if viper.GetBool("github.metaphor-pushed") {
		log.Info().Msgf("github.metaphor-pushed already executed, skipped")
		return nil
	}

	config := configs.ReadConfig()

	tfEntrypoint := config.GitOpsLocalRepoPath + "/terraform/github"
	if !skipMetaphor {
		err = os.Rename(fmt.Sprintf("%s/%s", tfEntrypoint, "metaphor-repos.md"), fmt.Sprintf("%s/%s", tfEntrypoint, "metaphor-repos.tf"))
		if err != nil {
			log.Error().Err(err).Msg("error renaming metaphor-repos.md to metaphor-repos.tf")
		}
	}

	err = os.Rename(fmt.Sprintf("%s/%s", tfEntrypoint, "remote-backend.md"), fmt.Sprintf("%s/%s", tfEntrypoint, "remote-backend.tf"))
	if err != nil {
		log.Error().Err(err).Msg("error renaming remote-backend.md to remote-backend.tf")
	}

	//this is not related with metaphor
	gitClient.PushLocalRepoUpdates(pkg.GitHubHost, gitHubOwner, "gitops", "github")
	terraform.InitMigrateApplyAutoApprove(dryRun, tfEntrypoint)
	if !skipMetaphor {
		log.Info().Msgf("Processing Repo: metaphor")
		repo.PrepareKubefirstTemplateRepo(
			dryRun,
			config,
			gitHubOwner,
			"metaphor",
			metaphorBranch,
			templateTag,
		)
		log.Info().Msgf("clone and detokenization of metaphor-template repository complete")

		gitClient.PushLocalRepoToEmptyRemote(pkg.GitHubHost, gitHubOwner, "metaphor", "github")
	}

	viper.Set("github.metaphor-pushed", true)
	err = viper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
}
