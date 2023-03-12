package k3d

import (
	"errors"
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst/internal/downloadManager"
	"github.com/rs/zerolog/log"
)

func DownloadTools(gitProvider, gitOwner string, toolsDir string) error {

	config := GetConfig(gitProvider, gitOwner)

	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		err := os.MkdirAll(toolsDir, os.ModePerm)
		if err != nil {
			log.Info().Msgf("%s directory already exists, continuing", toolsDir)
		}
	}

	//* k3d
	k3dDownloadUrl := fmt.Sprintf(
		"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
		K3dVersion,
		LocalhostOS,
		LocalhostARCH,
	)
	err := downloadManager.DownloadFile(config.K3dClient, k3dDownloadUrl)
	if err != nil {
		return errors.New(fmt.Sprintf("error while trying to download k3d: %s", err))
	}

	err = os.Chmod(config.K3dClient, 0755)
	if err != nil {
		return err
	}

	//* kubectl
	kubectlDownloadURL := fmt.Sprintf(
		"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
		KubectlVersion,
		LocalhostOS,
		LocalhostARCH,
	)

	err = downloadManager.DownloadFile(config.KubectlClient, kubectlDownloadURL)
	if err != nil {
		return errors.New(fmt.Sprintf("error while trying to download kubectl: %s", err))
	}

	err = os.Chmod(config.KubectlClient, 0755)
	if err != nil {
		return err
	}

	// * mkcert
	// https: //github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-darwin-amd64
	mkCertDownloadURL := fmt.Sprintf(
		"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
		MkCertVersion,
		MkCertVersion,
		LocalhostOS,
		LocalhostARCH,
	)

	err = downloadManager.DownloadFile(config.MkCertClient, mkCertDownloadURL)
	if err != nil {
		return errors.New(fmt.Sprintf("error while trying to download mkcert: %s", err))
	}
	err = os.Chmod(config.MkCertClient, 0755)
	if err != nil {
		return err
	}

	//* terraform
	terraformDownloadURL := fmt.Sprintf(
		"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
		TerraformVersion,
		TerraformVersion,
		LocalhostOS,
		LocalhostARCH,
	)
	zipPath := fmt.Sprintf("%s/terraform.zip", config.ToolsDir)

	err = downloadManager.DownloadZip(config.ToolsDir, terraformDownloadURL, zipPath)
	if err != nil {
		return errors.New(fmt.Sprintf("error while trying to download terraform: %s", err))
	}

	return nil
}
