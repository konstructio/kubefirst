package cmd

import (
	"fmt"
	"github.com/kubefirst/nebulous/configs"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		config := configs.ReadConfig()

		log.Println("removing $HOME/.kubefirst and $HOME/.flare")
		// todo ask for user input to verify?
		os.RemoveAll(fmt.Sprintf("%s/.kubefirst", config.HomePath))
		os.Remove(fmt.Sprintf("%s/.flare", config.HomePath))
		log.Println("removed $HOME/.kubefirst and $HOME/.flare")
		// todo log.Println("proceed to kubefirst create ")
		log.Println("proceed to flare nebulous create ")
	},
}

func init() {
	initCmd.AddCommand(cleanCmd)
}
