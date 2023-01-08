package downloadManager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/pkg"
)

// DownloadLocalTools - Download extra tools needed for local installations scenarios
func DownloadLocalTools(config *configs.Config) error {
	toolsDirPath := fmt.Sprintf("%s/tools", config.K1FolderPath)
	err := createDirIfDontExist(toolsDirPath)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChannel := make(chan error)
	wgDone := make(chan bool)
	wg.Add(2)

	go func() {
		// https://github.com/k3d-io/k3d/releases/download/v5.4.6/k3d-linux-amd64
		k3dDownloadURL := fmt.Sprintf(
			"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
			config.K3dVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		err = downloadFile(config.K3dPath, k3dDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}
		err = os.Chmod(config.K3dPath, 0755)
		if err != nil {
			errorChannel <- err
			return
		}
		wg.Done()
	}()

	go func() {
		// https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-darwin-amd64
		mkCertDownloadURL := fmt.Sprintf(
			"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
			config.MkCertVersion,
			config.MkCertVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		err = downloadFile(config.MkCertPath, mkCertDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}
		err = os.Chmod(config.MkCertPath, 0755)
		if err != nil {
			errorChannel <- err
			return
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		log.Info().Msg("download finished")
		return nil
	case err = <-errorChannel:
		close(errorChannel)
		return err
	}
}

// DownloadTools prepare download folder, and download the required installation tools for download. The downloads
// are done via Go routines, and error are handler by the error channel. In case of some error in the download process,
// the errorChannel will receive the error message, and process it back to the function caller.
func DownloadTools(config *configs.Config) error {

	log.Info().Msg("starting downloads...")
	toolsDirPath := fmt.Sprintf("%s/tools", config.K1FolderPath)

	// create folder if it doesn't exist
	err := createDirIfDontExist(toolsDirPath)
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
		kVersion := config.KubectlVersion
		if config.LocalOs == "darwin" && config.LocalArchitecture == "arm64" {
			kVersion = config.KubectlVersionM1
		}

		kubectlDownloadURL := fmt.Sprintf(
			"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
			kVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadURL)
		err = downloadFile(config.KubectlClientPath, kubectlDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}

		err = os.Chmod(config.KubectlClientPath, 0755)
		if err != nil {
			errorChannel <- err
			return
		}

		log.Info().Msgf("going to print the kubeconfig env in runtime: %s", os.Getenv("KUBECONFIG"))

		kubectlStdOut, kubectlStdErr, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "version", "--client", "--short")
		log.Info().Msgf("-> kubectl version:\n\t%s\n\t%s\n", kubectlStdOut, kubectlStdErr)
		if err != nil {
			errorChannel <- fmt.Errorf("failed to call kubectlVersionCmd.Run(): %v", err)
			return
		}
		wg.Done()
		log.Info().Msg("Kubectl download finished")
	}()

	go func() {

		// todo: adopt latest helmVersion := "v3.9.0"
		terraformVersion := config.TerraformVersion

		terraformDownloadURL := fmt.Sprintf(
			"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
			terraformVersion,
			terraformVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		log.Info().Msgf("Downloading terraform from %s", terraformDownloadURL)
		terraformDownloadZipPath := fmt.Sprintf("%s/tools/terraform.zip", config.K1FolderPath)
		err = downloadFile(terraformDownloadZipPath, terraformDownloadURL)
		if err != nil {
			errorChannel <- fmt.Errorf("error reading terraform file, %v", err)
			return
		}

		unzipDirectory := fmt.Sprintf("%s/tools", config.K1FolderPath)
		unzip(terraformDownloadZipPath, unzipDirectory)

		err = os.Chmod(unzipDirectory, 0777)
		if err != nil {
			errorChannel <- err
			return
		}

		err = os.Chmod(fmt.Sprintf("%s/terraform", unzipDirectory), 0755)
		if err != nil {
			errorChannel <- err
			return
		}
		err = os.RemoveAll(fmt.Sprintf("%s/terraform.zip", toolsDirPath))
		if err != nil {
			errorChannel <- err
			return
		}
		wg.Done()
		log.Info().Msg("Terraform download finished")
	}()

	go func() {
		helmVersion := config.HelmVersion
		helmDownloadURL := fmt.Sprintf(
			"https://get.helm.sh/helm-%s-%s-%s.tar.gz",
			helmVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		log.Info().Msgf("Downloading terraform from %s", helmDownloadURL)
		helmDownloadTarGzPath := fmt.Sprintf("%s/tools/helm.tar.gz", config.K1FolderPath)

		err = downloadFile(helmDownloadTarGzPath, helmDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}

		helmTarDownload, err := os.Open(helmDownloadTarGzPath)
		if err != nil {
			errorChannel <- fmt.Errorf("could not read helm download content")
			return
		}

		extractFileFromTarGz(
			helmTarDownload,
			fmt.Sprintf("%s-%s/helm", config.LocalOs, config.LocalArchitecture),
			config.HelmClientPath,
		)
		err = os.Chmod(config.HelmClientPath, 0755)
		if err != nil {
			errorChannel <- err
			return
		}

		os.Remove(helmDownloadTarGzPath)
		// currently argocd init values is generated by kubefirst ssh
		// todo helm install argocd --create-namespace --wait --values ~/.kubefirst/argocd-init-values.yaml argo/argo-cd
		helmStdOut, helmStdErr, err := pkg.ExecShellReturnStrings(
			config.HelmClientPath,
			"version",
			"--client",
			"--short",
		)
		if err != nil {
			log.Info().Msg(helmStdErr)
			errorChannel <- fmt.Errorf("error executing helm version command: %v", err)
			return
		}
		os.Remove(helmDownloadTarGzPath)

		log.Info().Msgf("Helm version: %s", helmStdOut)

		wg.Done()

		log.Info().Msg("Helm download finished")

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

// downloadFile Downloads a file from the "url" parameter, localFilename is the file destination in the local machine.
func downloadFile(localFilename string, url string) error {
	// create local file
	out, err := os.Create(localFilename)
	if err != nil {
		return err
	}
	defer out.Close()

	// get data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unable to download the required filed, the HTTP return status is: %s", resp.Status)
	}

	// writer the body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func extractFileFromTarGz(gzipStream io.Reader, tarAddress string, targetFilePath string) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Panic().Msg("extractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Panic().Msgf("extractTarGz: Next() failed: %s", err.Error())
		}
		log.Info().Msg(header.Name)
		if header.Name == tarAddress {
			switch header.Typeflag {
			case tar.TypeReg:
				outFile, err := os.Create(targetFilePath)
				if err != nil {
					log.Panic().Msgf("extractTarGz: Create() failed: %s", err.Error())
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					log.Panic().Msgf("extractTarGz: Copy() failed: %s", err.Error())
				}
				outFile.Close()

			default:
				log.Info().Msgf(
					"extractTarGz: uknown type: %s in %s\n",
					string(header.Typeflag),
					header.Name)
			}

		}
	}
}

func unzip(zipFilepath string, unzipDirectory string) error {
	dst := unzipDirectory
	archive, err := zip.OpenReader(zipFilepath)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		log.Info().Msgf("unzipping file %s", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return errors.New("invalid file path")
		}
		if f.FileInfo().IsDir() {
			log.Info().Msg("creating directory...")
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return err
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}

func createDirIfDontExist(toolsDirPath string) error {
	if _, err := os.Stat(toolsDirPath); errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(toolsDirPath, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}
