package cmd

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/kubefirst/kubefirst/internal/template"
	"github.com/spf13/cobra"
)

// templateCmd represents the template command
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Inform the template file (input) and rendered file (output)",
	Long:  `Based on template file, reading values from variables and k1rst file, another file is rendered using go-template`,
	Run: func(cmd *cobra.Command, args []string) {
		templateF, err := cmd.Flags().GetString("template")
		if err != nil {
			log.Println(err)
		}

		read, err := ioutil.ReadFile(templateF)
		if err != nil {
			panic(err)
		}

		renderedBuffer := template.Render(string(read))
		fmt.Print(renderedBuffer.String())
	},
}

func init() {
	rootCmd.AddCommand(templateCmd)

	templateCmd.Flags().String("template", "", "Template file (input)")

	err := templateCmd.MarkFlagRequired("template")
	if err != nil {
		log.Println(err)
		return
	}

}
