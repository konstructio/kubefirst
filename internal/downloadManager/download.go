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

	"github.com/rs/zerolog/log"
)

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
