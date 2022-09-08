package cmd

import (
	"log"
	"net/http"
	"github.com/spf13/cobra"
	sw "github.com/kubefirst/kubefirst/internal/api"
)

var api = &cobra.Command{
	Use:   "api",
	Short: "starts API server",
	Long:  "starts intermal API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("Server started")

		router := sw.NewRouter()
	
		log.Panic(http.ListenAndServe(":9095", router))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(api)
}
