package cmd

import (
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
	"log"
	"net/http"
)

var console = &cobra.Command{
	Use:   "console",
	Short: "starts ui server",
	Long:  "starts app server for the Kubefirst console",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := configs.ReadConfig()
		distFolder := fmt.Sprintf("%s/tools/console/dist", config.K1FolderPath)
		fileServer := http.FileServer(http.Dir(distFolder))
		http.Handle("/", fileServer)

		fmt.Printf("Starting server at port 8080\n")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatal(err)
    }
		return nil
	},
}

func init() {
	rootCmd.AddCommand(console)
}
