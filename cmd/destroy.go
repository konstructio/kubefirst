package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitClient"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// destroyCmd represents the destroy command
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "destroy Kubefirst management cluster",
	Long:  "destroy all the resources installed via Kubefirst installer",
	RunE: func(cmd *cobra.Command, args []string) error {

		//Destroy is implemented based on the flavor selected.
		if viper.GetString("cloud") == pkg.CloudAws {
			//just in case, we need downstream
			awsProfile := viper.GetString("aws.profile")
			os.Setenv("AWS_PROFILE", awsProfile)
			if viper.GetString("gitprovider") == gitClient.Github {
				err := destroyAwsGithubCmd.RunE(cmd, args)
				if err != nil {
					log.Println("Error destroying aws+github:", err)
					return err
				}

			} else if viper.GetString("gitprovider") == gitClient.Gitlab {
				err := destroyAwsGitlabCmd.RunE(cmd, args)
				if err != nil {
					log.Println("Error destroying aws+gitlab:", err)
					return err
				}
			} else {
				return fmt.Errorf("not supported git-provider")
			}

			log.Println("terraform base destruction complete")

		} else {
			return fmt.Errorf("not supported mode")
		}

		time.Sleep(time.Millisecond * 100)

		return nil
	},
}

func init() {
	clusterCmd.AddCommand(destroyCmd)
	currentCommand := destroyCmd
	flagset.DefineGlobalFlags(currentCommand)
	flagset.DefineDestroyFlags(currentCommand)
}
