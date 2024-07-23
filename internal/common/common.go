/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package common

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"io/ioutil"

	"github.com/kubefirst/kubefirst-api/pkg/configs"
	"github.com/kubefirst/kubefirst-api/pkg/docker"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/kubefirst/internal/cluster"
	"github.com/kubefirst/kubefirst/internal/launch"
	"github.com/kubefirst/kubefirst/internal/progress"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CheckResponse struct {
	// Current is current latest version on source.
	Current string

	// Outdate is true when target version is less than Curernt on source.
	Outdated bool

	// Latest is true when target version is equal to Current on source.
	Latest bool

	// New is true when target version is greater than Current on source.
	New bool
}

// CheckForVersionUpdate determines whether or not there is a new cli version available
func CheckForVersionUpdate() {
	if configs.K1Version != configs.DefaultK1Version {
		res, skip := versionCheck()
		if !skip {
			if res.Outdated {
				switch runtime.GOOS {
				case "darwin":
					fmt.Printf("A newer version (v%s) is available! Please upgrade with: \"brew update && brew upgrade kubefirst\"\n", res.Current)
				default:
					fmt.Printf("A newer version (v%s) is available! \"https://github.com/kubefirst/kubefirst/blob/main/build/README.md\"\n", res.Current)
				}
			}
		}
	}
}

// versionCheck compares local to remote version
func versionCheck() (res *CheckResponse, skip bool) {
	var latestVersion string
	flatVersion := strings.ReplaceAll(configs.K1Version, "v", "")

	resp, err := http.Get("https://raw.githubusercontent.com/Homebrew/homebrew-core/master/Formula/k/kubefirst.rb")

	if err != nil {
		fmt.Printf("checking for a newer version failed (cannot get Homebrew formula) with: %s", err)
		return nil, true
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("checking for a newer version failed (HTTP error) with: %s", err)
		return nil, true
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("checking for a newer version failed (cannot read the file) with: %s", err)
		return nil, true
	}

	bodyString := string(bodyBytes)
	if !strings.Contains(bodyString, "url \"https://github.com/kubefirst/kubefirst/archive/refs/tags/") {
		fmt.Printf("checking for a newer version failed (no reference to kubefirst release) with: %s", err)
		return nil, true
	}

	re := regexp.MustCompile(`.*/v(.*).tar.gz"`)
	matches := re.FindStringSubmatch(bodyString)
	latestVersion = matches[1]

	return &CheckResponse{
		Current:  flatVersion,
		Outdated: latestVersion < flatVersion,
		Latest:   latestVersion == flatVersion,
		New:      flatVersion > latestVersion,
	}, false
}

func GetRootCredentials(cmd *cobra.Command, args []string) error {
	clusterName := viper.GetString("flags.cluster-name")

	cluster, err := cluster.GetCluster(clusterName)
	if err != nil {
		progress.Error(err.Error())
		return err
	}

	progress.DisplayCredentials(cluster)

	return nil
}

func getgitmeta(clusterName string) (gitopsRepoName string, metaphorRepoName string) {
	var gitopsFound, metaphorFound bool
	homePath,err := os.UserHomeDir()
	dirs, err := ioutil.ReadDir(fmt.Sprintf("%s/.k1/%s", homePath, clusterName))
	if err != nil {
		log.Info().Msg("Error reading directory")
		return "cantfindgit","cantfindmeta"
	}

	for _, direc := range dirs {
		if direc.IsDir() {
			parentdir, err := ioutil.ReadDir(fmt.Sprintf("%s/.k1/%s/%s", homePath, clusterName, direc.Name()))
			if err != nil {
				log.Printf("Error reading directory %s: %v", direc.Name(), err)
				continue
			}

			for _, dir := range parentdir {
				if dir.IsDir() {
					if dir.Name() == "registry" {
						gitopsRepoName = direc.Name()
						gitopsFound = true
					} else
					if dir.Name() == ".github" {
						metaphorRepoName = direc.Name()
						metaphorFound = true
					}
				}
			}


		}
	}
	
	

	if !gitopsFound {
		log.Info().Msg("Gitops Repo not found")
		return "cantfindgit","cantfindmeta"
	}

	if !metaphorFound {
		log.Info().Msg("Metaphor Repo not found")
		os.Exit(1)
	}

	return gitopsRepoName, metaphorRepoName
}

func Destroy(cmd *cobra.Command, args []string) error {
	// Determine if there are active instal	ls
	gitProvider := viper.GetString("flags.git-provider")
	gitProtocol := viper.GetString("flags.git-protocol")
	cloudProvider := viper.GetString("kubefirst.cloud-provider")
	
	log.Info().Msg("destroying kubefirst platform")

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")

	gitopsRepoName,metaphorRepoName := getgitmeta(clusterName)

	// Switch based on git provider, set params
	cGitOwner := ""
	switch gitProvider {
	case "github":
		cGitOwner = viper.GetString("flags.github-owner")
	case "gitlab":
		cGitOwner = viper.GetString("flags.gitlab-owner")
	default:
		progress.Error("invalid git provider option")
	}

	// Instantiate aws config
	config := providerConfigs.GetConfig(
		clusterName,
		domainName,
		gitProvider,
		cGitOwner,
		gitProtocol,
		os.Getenv("CF_API_TOKEN"),
		os.Getenv("CF_ORIGIN_CA_ISSUER_API_TOKEN"),
		gitopsRepoName,
		metaphorRepoName,
	)

	progress.AddStep("Destroying k3d")

	launch.Down(true)

	progress.CompleteStep("Destroying k3d")
	progress.AddStep("Cleaning up environment")

	log.Info().Msg("resetting `$HOME/.kubefirst` config")
	viper.Set("argocd", "")
	viper.Set(gitProvider, "")
	viper.Set("components", "")
	viper.Set("kbot", "")
	viper.Set("kubefirst-checks", "")
	viper.Set("launch", "")
	viper.Set("kubefirst", "")
	viper.Set("flags", "")
	viper.Set("k1-paths", "")
	viper.WriteConfig()

	if _, err := os.Stat(config.K1Dir + "/kubeconfig"); !os.IsNotExist(err) {
		err = os.Remove(config.K1Dir + "/kubeconfig")
		if err != nil {
			progress.Error(fmt.Sprintf("unable to delete %q folder, error: %s", config.K1Dir+"/kubeconfig", err))
			return err
		}
	}

	progress.CompleteStep("Cleaning up environment")

	successMessage := `
###
#### :tada: Success` + "`Your k3d kubefirst platform has been destroyed.`" + `

### :blue_book: To delete a management cluster please see documentation:
https://docs.kubefirst.io/` + cloudProvider + `/deprovision
`

	progress.Success(successMessage)

	return nil
}

// checkDocker makes sure Docker is running before all commands
func CheckDocker(cmd *cobra.Command, args []string) {
	// Verify Docker is running
	dcli := docker.DockerClientWrapper{
		Client: docker.NewDockerClient(),
	}
	_, err := dcli.CheckDockerReady()
	if err != nil {
		progress.Error(fmt.Sprintf("Docker must be running to use this command. Error checking Docker status: %s", err))
	}
}
