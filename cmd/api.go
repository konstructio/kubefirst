package cmd

import (
	"log"
	"net/http"

	sw "github.com/kubefirst/kubefirst/internal/api"
	"github.com/spf13/cobra"
)

var api = &cobra.Command{
	Use:   "api",
	Short: "starts API server",
	Long:  "starts intermal API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("Server started")
		router := sw.NewRouter()

		//In case of error, we need to bubble it up
		return http.ListenAndServe("127.0.0.1:9095", router)
	},
}

func init() {
	rootCmd.AddCommand(api)
}
