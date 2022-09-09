package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/spf13/cobra"
)

var console = &cobra.Command{
	Use:   "console",
	Short: "starts ui server",
	Long:  "Starts UI where the user can see the credentials and all installed services",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := configs.ReadConfig()
		distFolder := fmt.Sprintf("%s/tools/console/dist", config.K1FolderPath)
		fileServer := http.FileServer(http.Dir(distFolder))
		http.Handle("/", fileServer)

		log.Printf("Starting server at port 9094\n")
		fmt.Printf("Starting server at port 9094\n")
		if err := http.ListenAndServe(":9094", nil); err != nil {
			log.Println(err)
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(console)
}
