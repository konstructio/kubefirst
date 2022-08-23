package cmd

import (
	"log"

	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Inform the template file (input) and rendered file (output)",
	Long:  `Based on template file, reading values from variables and k1rst file, another file is rendered using go-template`,
	Run: func(cmd *cobra.Command, args []string) {
		template, err := cmd.Flags().GetString("template")
		if err != nil {
			log.Println(err)
		}
		rendered, err := cmd.Flags().GetString("rendered")
		if err != nil {
			log.Println(err)
		}
		pkg.Template(template, rendered)
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.Flags().String("template", "", "Template file (input)")
	templateCmd.Flags().String("rendered", "", "Rendered file (output)")

	err := templateCmd.MarkFlagRequired("template")
	if err != nil {
		log.Println(err)
		return
	}

	err = templateCmd.MarkFlagRequired("rendered")
	if err != nil {
		log.Println(err)
		return
	}
}
