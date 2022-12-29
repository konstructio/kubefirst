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
	log.Info().Msg("starting checking tool downloads")

	err := createDirIfDontExist(config.K1ToolsPath)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errorChannel := make(chan error)
	wgDone := make(chan bool)
	wg.Add(2)

	go func() {
		_, err := os.Stat(config.K3dPath)
		if err == nil {
			log.Info().Msg("k3d already exists skiping download")
			wg.Done()
			return
		} else {
			log.Info().Msg("k3d binnary not found - starting download")

			k3dDownloadUrl := fmt.Sprintf(
				"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
				config.K3dVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			err = downloadFile(config.K3dPath, k3dDownloadUrl)
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
			log.Info().Msg("K3D download finished")

		}
	}()

	go func() {
		_, err := os.Stat(config.MkCertPath)
		if err == nil {
			log.Info().Msg("MkCert already exists skiping download")
			wg.Done()
			return
		} else {
			log.Info().Msg("MkCert binnary not found - starting download")

			mkCertDownloadUrl := fmt.Sprintf(
				"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
				config.MkCertVersion,
				config.MkCertVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			err = downloadFile(config.MkCertPath, mkCertDownloadUrl)
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
			log.Info().Msg("MkCert download finished")
		}
	}()

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		log.Info().Msg("finished tools download check")
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

	log.Info().Msg("starting checking tool downloads")

	// create folder if it doesn't exist
	err := createDirIfDontExist(config.K1ToolsPath)
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
		_, err := os.Stat(config.KubectlClientPath)
		if err == nil {
			log.Info().Msg("kubectl already exists skiping download")
			wg.Done()
			return
		} else {
			log.Info().Msg("kubectl binnary not found - starting download")

			kubectlDownloadUrl := fmt.Sprintf(
				"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
				config.KubectlVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadUrl)
			err = downloadFile(config.KubectlClientPath, kubectlDownloadUrl)
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
		}
	}()

	go func() {
		_, err := os.Stat(config.TerraformClientPath)
		if err == nil {
			log.Info().Msg("Terraform already exists skiping download")
			wg.Done()
			return
		} else {
			log.Info().Msg("Terraform binnary not found - starting download")

			terraformVersion := config.TerraformVersion

			terraformDownloadUrl := fmt.Sprintf(
				"https://releases.hashicorp.com/terraform/%s/terraform_%s_%s_%s.zip",
				terraformVersion,
				terraformVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			log.Info().Msgf("Downloading terraform from %s", terraformDownloadUrl)
			terraformDownloadZipPath := fmt.Sprintf("%s/terraform.zip", config.K1ToolsPath)
			err = downloadFile(terraformDownloadZipPath, terraformDownloadUrl)
			if err != nil {
				errorChannel <- fmt.Errorf("error reading terraform file, %v", err)
				return
			}

			unzip(terraformDownloadZipPath, config.K1ToolsPath)

			err = os.Chmod(config.K1ToolsPath, 0777)
			if err != nil {
				errorChannel <- err
				return
			}

			err = os.Chmod(fmt.Sprintf("%s/terraform", config.K1ToolsPath), 0755)
			if err != nil {
				errorChannel <- err
				return
			}
			err = os.RemoveAll(fmt.Sprintf("%s/terraform.zip", config.K1ToolsPath))
			if err != nil {
				errorChannel <- err
				return
			}
			wg.Done()
			log.Info().Msg("Terraform download finished")
		}
	}()

	go func() {
		_, err := os.Stat(config.HelmClientPath)
		if err == nil {
			log.Info().Msg("Helm already exists skiping download")
			wg.Done()
			return
		} else {
			log.Info().Msg("helm binnary not found - starting download")

			helmVersion := config.HelmVersion
			helmDownloadUrl := fmt.Sprintf(
				"https://get.helm.sh/helm-%s-%s-%s.tar.gz",
				helmVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			log.Info().Msgf("Downloading terraform from %s", helmDownloadUrl)
			helmDownloadTarGzPath := fmt.Sprintf("%s/helm.tar.gz", config.K1ToolsPath)

			err = downloadFile(helmDownloadTarGzPath, helmDownloadUrl)
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
		}
	}()

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		log.Info().Msg("finished tools download check")
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
