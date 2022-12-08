package civo

import (
	"log"

	"github.com/spf13/cobra"
)

func destroyCivo(cmd *cobra.Command, args []string) error {
	log.Println("running destroy civo github")

	// nextKubefirstDestroyCommand := "`kubefirst aws destroy"
	// nextKubefirstDestroyCommand = fmt.Sprintf("%s \n  --skip-tf-aws", nextKubefirstDestroyCommand)

	// config := configs.ReadConfig()

	// githubToken := config.GithubToken
	// if len(githubToken) == 0 {
	// 	return errors.New("ephemeral tokens not supported for cloud installations, please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
	// }
	// silentMode := false
	// dryRun := false
	// test it to remove .terraform and .terraform.lock resources before calling
	// executionControl := viper.GetBool("terraform.aws.apply.complete")
	// if executionControl {
	// 	pkg.InformUser("destroying github resources with terraform", silentMode)

	// 	tfEntrypoint := config.TerraformAwsEntrypointPath
	// 	err := terraform.InitDestroyAutoApprove(dryRun, config.TerraformAwsEntrypointPath)
	// 	if err != nil {
	// 		log.Printf("error executing terraform destroy %s", tfEntrypoint)
	// 		return err
	// 	}

	// 	viper.Set("terraform.aws.apply.complete", false)
	// 	viper.WriteConfig()

	// 	pkg.InformUser(fmt.Sprintf("destroy github repositories in github.com/%s", viper.GetString("github.owner")), silentMode)
	// 	// progressPrinter.IncrementTracker("step-github", 1)
	// } else {
	// 	log.Println("already destroyed aws terraform resources - continuing")
	// }

	// //* this condition needs to include ECR somehow
	// //* we should be checking if apply happened but in some circumstances we need
	// //* to destroy github even if the repo's didnt create to remove the ecr registries
	// executionControl = viper.GetBool("terraform.github.apply.complete")
	// if executionControl {
	// 	pkg.InformUser("destroying github resources with terraform", silentMode)

	// 	tfEntrypoint := config.TerraformGithubEntrypointPath
	// 	err := terraform.InitDestroyAutoApprove(dryRun, config.TerraformGithubEntrypointPath)
	// 	if err != nil {
	// 		log.Printf("error executing terraform destroy %s", tfEntrypoint)
	// 		return err
	// 	}

	// 	viper.Set("terraform.github.apply.complete", false)
	// 	viper.WriteConfig()

	// 	pkg.InformUser(fmt.Sprintf("destroy github repositories at https://github.com/%s", viper.GetString("github.owner")), silentMode)
	// 	// progressPrinter.IncrementTracker("step-github", 1)
	// } else {
	// 	log.Println("already destroyed github terraform resources - continuing")
	// }

	return nil
}
