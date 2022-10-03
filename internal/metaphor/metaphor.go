package metaphor

import (
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/spf13/viper"
)

// DeployMetaphorGitlab - Deploy metaphor applications on gitlab install
func DeployMetaphorGitlab(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, DeployMetaphorGitlab skipped.")
		return nil
	}
	log.Printf("cloning and detokenizing the metaphor-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of metaphor-template repository complete")

	log.Printf("cloning and detokenizing the metaphor-go-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor-go", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of metaphor-go-template repository complete")

	log.Printf("cloning and detokenizing the metaphor-frontend-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "metaphor-frontend", viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of metaphor-frontend-template repository complete")

	if !viper.GetBool("gitlab.metaphor-pushed") {
		log.Println("Pushing metaphor repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor")
		viper.Set("gitlab.metaphor-pushed", true)
		viper.WriteConfig()
		log.Println("clone and detokenization of metaphor-frontend-template repository complete")
	}

	// Go template
	if !viper.GetBool("gitlab.metaphor-go-pushed") {
		log.Println("Pushing metaphor-go repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor-go")
		viper.Set("gitlab.metaphor-go-pushed", true)
		viper.WriteConfig()
	}

	// Frontend template
	if !viper.GetBool("gitlab.metaphor-frontend-pushed") {
		log.Println("Pushing metaphor-frontend repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "metaphor-frontend")
		viper.Set("gitlab.metaphor-frontend-pushed", true)
		viper.WriteConfig()
	}
	return nil
}

// DeployMetaphorGithub - Deploy metaphor applications on github install
func DeployMetaphorGithub(globalFlags flagset.GlobalFlags) error {
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, DeployMetaphorGithub skipped.")
		return nil
	}
	githubOwner := viper.GetString("github.owner")
	githubHost := viper.GetString("github.host")
	if viper.GetBool("github.metaphor-pushed") {
		log.Println("github.metaphor-pushed already executed, skipped")
		return nil
	}
	config := configs.ReadConfig()
	tfEntrypoint := "github"
	directory := fmt.Sprintf("%s/gitops/terraform/%s", config.K1FolderPath, tfEntrypoint)
	err := os.Rename(fmt.Sprintf("%s/%s", directory, "metaphor-repos.md"), fmt.Sprintf("%s/%s", directory, "metaphor-repos.tf"))
	if err != nil {
		log.Println("error renaming metaphor-repos.md to metaphor-repos.tf", err)
	}
	gitClient.PushLocalRepoUpdates(githubHost, githubOwner, "gitops", "github")
	terraform.InitApplyAutoApprove(globalFlags.DryRun, directory, tfEntrypoint)

	repos := [3]string{"metaphor", "metaphor-go", "metaphor-frontend"}
	for _, element := range repos {
		log.Println("Processing Repo:", element)
		repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), element, viper.GetString("metaphor.branch"), viper.GetString("template.tag"))
		log.Printf("clone and detokenization of %s-template repository complete", element)
		githubHost := viper.GetString("github.host")

		gitClient.PushLocalRepoToEmptyRemote(githubHost, githubOwner, element, "github")

	}

	viper.Set("github.metaphor-pushed", true)
	viper.WriteConfig()
	return nil
}
