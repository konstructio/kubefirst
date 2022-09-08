/*
 * Swagger API for Kubefirst - OpenAPI 3.0
 *
 * Sample API for Kubefirst UI - Local
 *
 * API version: 0.0.1
 * Contact: kray@kubefrist.com
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package swagger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/viper"
)

func ConfigsGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.WriteHeader(http.StatusOK)

	configValues := []Config{
		{
			Key:   "HOSTED_ZONE_NAME",
			Value: viper.GetString("aws.hostedzonename"),
		},
		{
			Key:   "ARGO_URL",
			Value: fmt.Sprintf("https://argo.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "ARGOCD_USERNAME",
			Value: viper.GetString("argocd.admin.username"),
		},
		{
			Key:   "ARGOCD_PASSWORD",
			Value: viper.GetString("argocd.admin.password"),
		},
		{
			Key:   "ARGOCD_URL",
			Value: fmt.Sprintf("https://argocd.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "GITLAB_URL",
			Value: fmt.Sprintf("https://gitlab.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "GITLAB_USERNAME",
			Value: "root",
		},
		{
			Key:   "GITLAB_PASSWORD",
			Value: viper.GetString("gitlab.root.password"),
		},
		{
			Key:   "VAULT_URL",
			Value: fmt.Sprintf("https://vault.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "VAULT_TOKEN",
			Value: viper.GetString("vault.token"),
		},
		{
			Key:   "ATLANTIS_URL",
			Value: fmt.Sprintf("https://atlantis.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "ADMIN_EMAIL",
			Value: viper.GetString("adminemail"),
		},
		{
			Key:   "METAPHOR_DEV",
			Value: fmt.Sprintf("https://metaphor-development.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_GO_DEV",
			Value: fmt.Sprintf("https://metaphor-go-development.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_FRONT_DEV",
			Value: fmt.Sprintf("https://metaphor-frontend-development.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_STAGING",
			Value: fmt.Sprintf("https://metaphor-staging.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_GO_STAGING",
			Value: fmt.Sprintf("https://metaphor-go-staging.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_FRONT_STAGING",
			Value: fmt.Sprintf("https://metaphor-frontend-staging.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_PROD",
			Value: fmt.Sprintf("https://metaphor-production.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_GO_PROD",
			Value: fmt.Sprintf("https://metaphor-go-production.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "METAPHOR_FRONT_PROD",
			Value: fmt.Sprintf("https://metaphor-frontend-production.%s", viper.GetString("aws.hostedzonename")),
		},
		{
			Key:   "ADMIN_EMAIL",
			Value: viper.GetString("adminemail"),
		},
	}

	jsonData, err := json.Marshal(configValues)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		return
	}

	_, err = w.Write(jsonData)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
}
