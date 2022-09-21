package ciTools

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/spf13/viper"
)

// DeployGitlab - Deploy CI applications on gitlab install
func DeployGitlab(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, DeployGitlab skipped.")
		return nil
	}
	log.Printf("cloning and detokenizing the ci-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "ci", viper.GetString("ci.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of ci-template repository complete")

	if !viper.GetBool("gitlab.ci-pushed") {
		log.Println("Pushing ci repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "ci")
		viper.Set("gitlab.ci-pushed", true)
		viper.WriteConfig()
		log.Println("clone and detokenization of ci-template repository complete")
	}

	return nil
}
