package chartMuseum

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/internal/httpCommon"
	"github.com/spf13/viper"
)

// IsChartMuseumReady - check is current instance of ChartMuseum is ready to receive deployments
// refers to: https://github.com/kubefirst/kubefirst/issues/386
func IsChartMuseumReady() (bool, error) {
	// todo local uses a different function pkg.AwaitHostNTimes
	url := fmt.Sprintf("https://chartmuseum.%s/index.yaml", viper.GetString("aws.hostedzonename"))

	response, err := httpCommon.CustomHttpClient(false).Get(url)
	//not ready, should result on exit 1
	if err != nil {
		log.Warn().Msgf("error: ChartMuseum is not ready: %s", err)
		return false, err
	}

	log.Info().Msgf("ChartMuseum check returns: %d", response.StatusCode)
	//Add some check to see if the yaml is "valid"
	//Usual payload, it:
	/*
		entries: {}
		generated: "2022-09-19T19:32:30Z"
		serverInfo: {}

	*/
	//For now, what works is already enough.
	if response.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil

}
