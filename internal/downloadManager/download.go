package downloadManager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
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

type KubectlVersion struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:clientVersion`
}

type TerraformVersion struct {
	TerraformVersion string `json:"terraform_version"`
}

// DownloadFile Downloads a file from the "url" parameter, localFilename is the file destination in the local machine.
func DownloadFile(localFilename string, url string) error {
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
		return fmt.Errorf("unable to download the required file, the HTTP return status is: %s", resp.Status)
	}

	// writer the body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func ExtractFileFromTarGz(gzipStream io.Reader, tarAddress string, targetFilePath string) {
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

func Unzip(zipFilepath string, unzipDirectory string) error {
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

func DownloadTarGz(binaryPath string, tarAddress string, targzPath string, URL string) error {

	log.Info().Msgf("Downloading tar.gz from %s", URL)

	err := DownloadFile(targzPath, URL)
	if err != nil {
		return err
	}

	tarContent, err := os.Open(targzPath)
	if err != nil {
		return err
	}

	ExtractFileFromTarGz(
		tarContent,
		tarAddress,
		binaryPath,
	)
	os.Remove(targzPath)
	err = os.Chmod(binaryPath, 0755)
	if err != nil {
		return err
	}
	return nil
}

func DownloadZip(toolsDir string, URL string, zipPath string) error {

	log.Info().Msgf("Downloading zip from %s", "URL")

	err := DownloadFile(zipPath, URL)
	if err != nil {
		return err
	}

	err = Unzip(zipPath, toolsDir)
	if err != nil {
		return err
	}

	err = os.RemoveAll(zipPath)
	if err != nil {
		return err
	}

	return nil
}

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
		// https://github.com/k3d-io/k3d/releases/download/v5.4.6/k3d-linux-amd64
		k3dDownloadURL := fmt.Sprintf(
			"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
			config.K3dVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		err = DownloadFile(config.K3dPath, k3dDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}
		if FileExists(config.K3dPath) && VerifyCacheK3D(config) {
			log.Info().Msg("K3d exists and cache validated - skipping download")
			wg.Done()
			return
		}

		if FileExists(config.K3dPath) && !VerifyCacheK3D(config) {
			log.Info().Msg("K3d exists and cache version is invalid - continuing...")
			log.Info().Msg("divergent versions may not work well, we recommend running `kubefirst clean` command and try again")
			wg.Done()
			return
		}

		if !FileExists(config.K3dPath) {
			log.Info().Msg("k3d binnary not found - starting download")

			k3dDownloadUrl := fmt.Sprintf(
				"https://github.com/k3d-io/k3d/releases/download/%s/k3d-%s-%s",
				config.K3dVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			err = DownloadFile(config.K3dPath, k3dDownloadUrl)
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
		// https://github.com/FiloSottile/mkcert/releases/download/v1.4.4/mkcert-v1.4.4-darwin-amd64
		mkCertDownloadURL := fmt.Sprintf(
			"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
			config.MkCertVersion,
			config.MkCertVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		err = DownloadFile(config.MkCertPath, mkCertDownloadURL)
		if err != nil {
			errorChannel <- err
			if FileExists(config.MkCertPath) && VerifyCacheMkCert(config) {
				log.Info().Msg("MkCert exists and cache validated - skipping download")
				wg.Done()
				return
			}

			if FileExists(config.MkCertPath) && !VerifyCacheMkCert(config) {
				log.Info().Msg("MkCert exists and cache version is invalid - continuing...")
				log.Info().Msg("divergent versions may not work well, we recommend running `kubefirst clean` command and try again")
				wg.Done()
				return
			}

			if !FileExists(config.MkCertPath) {
				log.Info().Msg("MkCert binnary not found - starting download")

				mkCertDownloadUrl := fmt.Sprintf(
					"https://github.com/FiloSottile/mkcert/releases/download/%s/mkcert-%s-%s-%s",
					config.MkCertVersion,
					config.MkCertVersion,
					config.LocalOs,
					config.LocalArchitecture,
				)
				err = DownloadFile(config.MkCertPath, mkCertDownloadUrl)
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

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func VerifyCacheK3D(config *configs.Config) bool {
	log.Debug().Msg("K3D checking version")
	out, _, _ := pkg.ExecShellReturnStrings(config.K3dPath, "version")
	matchVersion := strings.Contains(out, fmt.Sprintf("k3d version %s", config.K3dVersion))
	if matchVersion {
		log.Debug().Msgf("K3d version matched (%s), returning true", config.K3dVersion)
		return true
	}

	log.Debug().Msgf("K3d version does not match (%s), returning false", config.K3dVersion)
	return false
}

func VerifyCacheMkCert(config *configs.Config) bool {
	log.Debug().Msg("MkCert checking version")
	out, _, _ := pkg.ExecShellReturnStrings(config.MkCertPath, "--version")
	trimed := strings.TrimSpace(out)
	if trimed == config.MkCertVersion {
		log.Debug().Msgf("MkCert version matched (%s/%s), returning true", trimed, config.MkCertVersion)
		return true
	}
	log.Debug().Msgf("MkCert version does not match (%s/%s), returning false", trimed, config.MkCertVersion)
	return false
}

func VerifyCacheTerraform(config *configs.Config) bool {
	log.Debug().Msg("Terraform checking version")
	out, _, _ := pkg.ExecShellReturnStrings(config.TerraformClientPath, "version", "-json")
	data := []byte(out)
	var tfVersion TerraformVersion
	err := json.Unmarshal(data, &tfVersion)
	if err != nil {
		log.Error().Err(err).Msgf("Error unmarshal: %s", err)
	}

	if tfVersion.TerraformVersion == config.TerraformVersion {
		log.Debug().Msgf("Terraform version matched (%s/%s), returning true", tfVersion.TerraformVersion, config.TerraformVersion)
		return true
	}

	log.Debug().Msgf("Terraform version does not match (%s/%s), returning false", tfVersion.TerraformVersion, config.TerraformVersion)
	return false
}

func VerifyCacheKubectl(config *configs.Config) bool {
	log.Debug().Msg("Kubectl checking version")
	out, _, _ := pkg.ExecShellReturnStrings(config.KubectlClientPath, "version", "-o", "json")
	data := []byte(out)
	var k KubectlVersion
	err := json.Unmarshal(data, &k)
	if err != nil {
		log.Error().Err(err).Msgf("Error unmarshal: %s", err)
	}

	if k.ClientVersion.GitVersion == config.KubectlVersion {
		log.Debug().Msgf("Kubectl version matched (%s/%s), returning true", k.ClientVersion.GitVersion, config.KubectlVersion)
		return true
	}

	log.Debug().Msgf("Kubectl version does not match (%s/%s), returning false", k.ClientVersion.GitVersion, config.KubectlVersion)
	return false
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
		kVersion := config.KubectlVersion
		if config.LocalOs == "darwin" && config.LocalArchitecture == "arm64" {
			kVersion = config.KubectlVersion
		}

		kubectlDownloadURL := fmt.Sprintf(
			"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
			kVersion,
			config.LocalOs,
			config.LocalArchitecture,
		)
		log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadURL)
		err = DownloadFile(config.KubectlClientPath, kubectlDownloadURL)
		if err != nil {
			errorChannel <- err
			return
		}
		if FileExists(config.KubectlClientPath) && VerifyCacheKubectl(config) {
			log.Info().Msg("kubectl exists and cache validated - skipping download")
			wg.Done()
			return
		}

		if FileExists(config.KubectlClientPath) && !VerifyCacheKubectl(config) {
			log.Info().Msg("kubectl exists and cache version is invalid - continuing...")
			log.Info().Msg("divergent versions may not work well, we recommend running `kubefirst clean` command and try again")
			wg.Done()
			return
		}

		if !FileExists(config.KubectlClientPath) {
			log.Info().Msg("kubectl not found - starting download")

			kubectlDownloadUrl := fmt.Sprintf(
				"https://dl.k8s.io/release/%s/bin/%s/%s/kubectl",
				config.KubectlVersion,
				config.LocalOs,
				config.LocalArchitecture,
			)
			log.Info().Msgf("Downloading kubectl from: %s", kubectlDownloadUrl)
			err = DownloadFile(config.KubectlClientPath, kubectlDownloadUrl)
			if err != nil {
				errorChannel <- err
				return
			}

			err = os.Chmod(config.KubectlClientPath, 0755)
			if err != nil {
				errorChannel <- err
				return
			}

			verifyBin := VerifyCacheKubectl(config)
			if !verifyBin {
				errorChannel <- fmt.Errorf("failed to verify kubectl download: %v", err)
				return
			}
			wg.Done()
			log.Info().Msg("Kubectl download finished")
		}
	}()

	go func() {

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
		err = DownloadFile(terraformDownloadZipPath, terraformDownloadURL)
		if err != nil {
			errorChannel <- fmt.Errorf("error reading terraform file, %v", err)
			return
		}
		if FileExists(config.TerraformClientPath) && VerifyCacheTerraform(config) {
			log.Info().Msg("Terraform exists and cache validated - skipping download")
			wg.Done()
			return
		}

		if FileExists(config.TerraformClientPath) && !VerifyCacheTerraform(config) {
			log.Info().Msg("Terraform exists and cache version is invalid - continuing...")
			log.Info().Msg("divergent versions may not work well, we recommend running `kubefirst clean` command and try again")
			wg.Done()
			return
		}

		if !FileExists(config.TerraformClientPath) {
			log.Info().Msg("Terraform not found - starting download")

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
			err = DownloadFile(terraformDownloadZipPath, terraformDownloadUrl)
			if err != nil {
				errorChannel <- fmt.Errorf("error reading terraform file, %v", err)
				return
			}

			Unzip(terraformDownloadZipPath, config.K1ToolsPath)

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
