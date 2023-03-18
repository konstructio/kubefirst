package aws

import (
	"fmt"
	"os"
	"sync"

	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
)

func DownloadTools(awsConfig *AwsConfig, kubectlClientVersion string, terraformClientVersion string) error {

	log.Info().Msg("starting downloads...")

	// create folder if it doesn't exist
	err := pkg.CreateDirIfNotExist(awsConfig.ToolsDir)
	if err != nil {
		return err
	}

	errorChannel := make(chan error)
	wgDone := make(chan bool)
	// create a waiting group (translating: create a queue of functions, and only pass the wg.Wait() function down
	// bellow after all the wg.Add(3) functions are done (wg.Done)
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {

		kubectlDownloadURL := fmt.Sprintf(
			"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
			kubectlClientVersion,
			pkg.LocalhostOS,
			pkg.LocalhostARCH,
		)
		log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadURL)
		err = downloadManager.DownloadFile(awsConfig.KubectlClient, kubectlDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}

		err = os.Chmod(awsConfig.KubectlClient, 0755)
		if err != nil {
			errorChannel <- err
			return
		}

		log.Info().Msgf("going to print the kubeconfig env in runtime: %s", os.Getenv("KUBECONFIG"))

		kubectlStdOut, kubectlStdErr, err := pkg.ExecShellReturnStrings(awsConfig.KubectlClient, "version", "--client", "--short")
		log.Info().Msgf("-> kubectl version:\n\t%s\n\t%s\n", kubectlStdOut, kubectlStdErr)
		if err != nil {
			errorChannel <- fmt.Errorf("failed to call kubectlVersionCmd.Run(): %v", err)
			return
		}
		wg.Done()
		log.Info().Msg("Kubectl download finished")
	}()

	go func() {

		terraformDownloadURL := fmt.Sprintf(
			"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
			terraformClientVersion,
			terraformClientVersion,
			pkg.LocalhostOS,
			pkg.LocalhostARCH,
		)
		log.Info().Msgf("Downloading terraform from %s", terraformDownloadURL)
		terraformDownloadZipPath := fmt.Sprintf("%s/terraform.zip", awsConfig.ToolsDir)
		err = downloadManager.DownloadFile(terraformDownloadZipPath, terraformDownloadURL)
		if err != nil {
			errorChannel <- fmt.Errorf("error downloading terraform file, %v", err)
			return
		}

		downloadManager.Unzip(terraformDownloadZipPath, awsConfig.ToolsDir)

		err = os.Chmod(awsConfig.ToolsDir, 0777)
		if err != nil {
			errorChannel <- err
			return
		}

		err = os.Chmod(fmt.Sprintf("%s/terraform", awsConfig.ToolsDir), 0755)
		if err != nil {
			errorChannel <- err
			return
		}
		err = os.RemoveAll(fmt.Sprintf("%s/terraform.zip", awsConfig.ToolsDir))
		if err != nil {
			errorChannel <- err
			return
		}
		// todo output terraform client version to be consistent with others
		wg.Done()
		log.Info().Msg("Terraform download finished")
	}()

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		log.Info().Msg("downloads finished")
		return nil
	case err = <-errorChannel:
		close(errorChannel)
		return err
	}
}
