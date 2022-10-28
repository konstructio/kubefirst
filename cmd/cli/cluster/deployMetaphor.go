package cluster

import (
	"log"
	"time"

	"github.com/kubefirst/kubefirst/internal/metaphor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployMetaphorDryRun bool

// deployMetaphorCommand represents the deployMetaphor command
func deployMetaphorCommand() *cobra.Command {
	deployMetaphorCmd := &cobra.Command{
		Use:   "deploy-metaphor",
		Short: "Add metaphor applications to the cluster",
		Long:  `TBD`,
		RunE:  runDeployMetaphorCmd,
	}
	deployMetaphorCmd.Flags().BoolVar(&deployMetaphorDryRun, "dry-run", false, "")
	return deployMetaphorCmd
}

func runDeployMetaphorCmd(cmd *cobra.Command, args []string) error {

	log.Println("deployMetaphor called")
	start := time.Now()
	defer func() {
		//The goal of this code is to track execution time
		duration := time.Since(start)
		log.Printf("[000] deploy-metaphor duration is %s", duration)

	}()

	if viper.GetBool("option.metaphor.skip") {
		log.Println("[99] Deployment of metpahor microservices skiped")
		return nil
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return err
	}
	if viper.GetString("gitprovider") == "github" {
		return metaphor.DeployMetaphorGithub(dryRun)
	} else {
		return metaphor.DeployMetaphorGitlab(dryRun)
	}
}
