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
	Long:  "starts internal API server that is consumed by the console UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("Console UI API started")
		router := sw.NewRouter()

		//In case of error, we need to bubble it up
		return http.ListenAndServe(":9095", router)
	},
}

func init() {
	rootCmd.AddCommand(api)
}
