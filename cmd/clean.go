/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
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
		// todo delete the s3 buckets associated with the ~/.flare file
		// todo ask for user input to verify deletion?
		log.Println("removing $HOME/.kubefirst and $HOME/.flare")
		os.RemoveAll(fmt.Sprintf("%s/.kubefirst", home))
		os.Remove(fmt.Sprintf("%s/.flare", home))
		log.Println("removed $HOME/.kubefirst and $HOME/.flare")
		if err := os.Mkdir(fmt.Sprintf("%s/.kubefirst", home), os.ModePerm); err != nil {
			log.Panicf("error: could not create directory $HOME/.kubefirst - it must exist to continue %s", err)
		}
		toolsDir := fmt.Sprintf("%s/.kubefirst/tools", home)
		if err := os.Mkdir(toolsDir, os.ModePerm); err != nil {
			log.Panicf("error: could not create directory $HOME/.kubefirst/tools - it must exist to continue %s", err)
		}

		log.Println("created $HOME/.kubefirst and $HOME/.kubefirst/tools - proceed to `kubefirst init`")
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
