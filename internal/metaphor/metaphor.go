package metaphor

import (
	"log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/internal/githubWrapper"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/spf13/viper"
)

func DeployMetaphorGitlab(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	log.Printf("cloning and detokenizing the metaphor-template repository")
	repo.PrepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor", "", viper.GetString("template.tag"))
	log.Println("clone and detokenization of metaphor-template repository complete")

	log.Printf("cloning and detokenizing the metaphor-go-template repository")
	repo.PrepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor-go", "", viper.GetString("template.tag"))
	log.Println("clone and detokenization of metaphor-go-template repository complete")

	log.Printf("cloning and detokenizing the metaphor-frontend-template repository")
	repo.PrepareKubefirstTemplateRepo(config, viper.GetString("gitops.owner"), "metaphor-frontend", "", viper.GetString("template.tag"))
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

func DeployMetaphorGithub(globalFlags flagset.GlobalFlags) error {
	owner := viper.GetString("github.owner")

	if viper.GetBool("github.metaphor-pushed") {
		log.Println("github.metaphor-pushed already executed, skiped")
		return nil
	}

	gitWrapper := githubWrapper.New()
	//repos := [2]string{"metaphor-go", "metaphor-frontend"}
	repos := [3]string{"metaphor", "metaphor-go", "metaphor-frontend"}
	for _, element := range repos {
		log.Printf("Processing Repo:", element)
		gitWrapper.CreatePrivateRepo(viper.GetString("github.org"), element, "Kubefirst"+element)
		directory, err := gitClient.CloneRepoAndDetokenizeTemplate("kubefirst", element, element, "", viper.GetString("template.tag"))
		if err != nil {
			log.Printf("Error clonning and detokizing repo %s", "metaphor")
			return err
		}
		gitClient.PopulateRepoWithToken(owner, element, directory, viper.GetString("github.host"))
	}

	viper.Set("github.metaphor-pushed", true)
	viper.WriteConfig()
	return nil
}
