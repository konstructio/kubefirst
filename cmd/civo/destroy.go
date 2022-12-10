package civo

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyCivo(cmd *cobra.Command, args []string) error {
	log.Println("running destroy for civo kubefirst installation")

	// nextKubefirstDestroyCommand := "`kubefirst aws destroy"
	// nextKubefirstDestroyCommand = fmt.Sprintf("%s \n  --skip-tf-aws", nextKubefirstDestroyCommand)

	config := configs.ReadConfig()

	githubToken := config.GithubToken
	if len(githubToken) == 0 {
		return errors.New("ephemeral tokens not supported for cloud installations, please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
	}
	// todo with these two..
	silentMode := false
	dryRun := false
	if !viper.GetBool("terraform.github.apply.complete") || viper.GetBool("terraform.github.destroy.complete") {
		pkg.InformUser("destroying github resources with terraform", silentMode)

		tfEntrypoint := config.GitOpsRepoPath + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = terraform.GithubTerraformEnvs(tfEnvs)
		err := terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("terraform.github.apply.complete", false)
		viper.Set("terraform.github.destroy.complete", true)
		viper.WriteConfig()
		pkg.InformUser("github resources terraform destroyed", silentMode)
	}

	//* successful cleanup of resources means we can clean up
	//* the ~/.k1/gitops so we can re-excute a `rebuild gitops` which would allow us
	//* to iterate without re-downloading etc

	if viper.GetBool("terraform.github.apply.complete") {

		// delete files and folders
		err := os.RemoveAll(config.K1FolderPath + "/gitops")
		if err != nil {
			return fmt.Errorf("unable to delete %q folder, error is: %s", config.K1FolderPath+"/gitops", err)
		}

		err = os.Remove(config.KubefirstConfigFilePath)
		if err != nil {
			return fmt.Errorf("unable to delete %q file, error is: ", err)
		}
		// re-create .kubefirst file
		kubefirstFile, err := os.Create(config.KubefirstConfigFilePath)
		if err != nil {
			return fmt.Errorf("error: could not create `$HOME/.kubefirst` file: %v", err)
		}
		err = kubefirstFile.Close()
		if err != nil {
			return err
		}

		viper.Set("template-repo.gitops.removed", true)
		viper.WriteConfig()
	}

	return nil
}
