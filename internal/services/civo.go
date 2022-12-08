package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kubefirst/kubefirst/pkg"
)

type CivoService struct {
	httpClient pkg.HTTPDoer
}

type CivoCluster struct {
	Name string `json:name`
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

// ListKubernetesClusters lists the kubernetes clusters in the account
// todo break out kubefirst check
func (service CivoService) ListKubernetesClusters(civoApiKey string) error {

	req, err := http.NewRequest(http.MethodGet, "https://api.civo.com/v2/kubernetes/clusters", nil)
	if err != nil {
		log.Println("error setting request")
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", civoApiKey))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return errors.New("unable to authenticate with the civo")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.New("error reading body")
	}

	var civoClusterList CivoClusterList
	err = json.Unmarshal(body, &civoClusterList)
	if err != nil {
		log.Println(err)
	}
	if len(civoClusterList.Items) != 0 {
		for _, cluster := range civoClusterList.Items {
			if cluster.Name == "kubefirst" {
				return errors.New("a cluster with the name `kubefirst` already exists,\nplease provide the --cluster-name flag")
			}
		}
	}

	return nil
}
