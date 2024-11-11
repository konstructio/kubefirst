package progress

import (
	"fmt"
	"time"

	"github.com/konstructio/kubefirst/internal/cluster"
)

func WatchClusterForCi(clusterName string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				provisioningCluster, _ := cluster.GetCluster(clusterName)

				if provisioningCluster.Status == "error" {
					fmt.Printf("unable to provision cluster: %s", provisioningCluster.LastCondition)
					done <- true
				}

				if provisioningCluster.Status == "provisioned" {
					fmt.Println("\n cluster has been provisioned via ci")
					fmt.Printf("\n kubefirst URL: https://kubefirst.%s \n", provisioningCluster.DomainName)
					done <- true
				}
			}
		}
	}()

	<-done
}
