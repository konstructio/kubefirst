package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/kubefirst/kubefirst/pkg"
)

type CivoService struct {
	httpClient pkg.HTTPDoer
}

type CivoCluster struct {
	Name string `json:"name"`
}

type CivoClusterList struct {
	Items []CivoCluster `json:"items"`
}

// NewCivoService instantiate a new GitHub service
func NewCivoService(httpClient pkg.HTTPDoer) *CivoService {
	return &CivoService{
		httpClient: httpClient,
	}
}

// ListKubernetesClusters is used to verify authentication of the provided CIVO_TOKEN value
func (service CivoService) ListKubernetesClusters(civoApiKey, clusterName string) (int, error) {

	req, err := http.NewRequest(http.MethodGet, "https://api.civo.com/v2/kubernetes/clusters", nil)
	if err != nil {
		log.Println("error setting request")
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", civoApiKey))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != http.StatusOK {
		return 0, errors.New("unable to authenticate with the civo")
	}

	return res.StatusCode, nil

}
