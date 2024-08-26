package progress

import (
	"fmt"
	"log"
	"time"

	"github.com/kubefirst/kubefirst/internal/cluster"
)

func WatchClusterForCi(clusterName string) {
	// Checks cluster status every 10 seconds
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
					log.Fatalf("unable to provision cluster: %s", provisioningCluster.LastCondition)
					done <- true
				}

				if provisioningCluster.Status == "provisioned" {
					fullDomainName := provisioningCluster.DomainName

					if provisioningCluster.SubdomainName != "" {
						fullDomainName = fmt.Sprintf("%s.%s", provisioningCluster.SubdomainName, provisioningCluster.DomainName)
					}

					fmt.Println("\n cluster has been provisioned via ci")
					fmt.Printf("\n kubefirst URL: https://kubefirst.%s \n", fullDomainName)
					done <- true
				}
			}
		}
	}()

	// waits until the provision is done
	<-done
}
