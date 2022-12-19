package gitClient

import (
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// AppendFile verify if a file must be appended to commited gitops
// meant to help exclude undesired state files to be pushed to gitops
func AppendFile(cloudType string, reponame string, filename string) bool {
	//result := true
	//TODO: make this to be loaced by Arrays of exclusion rules
	//TODO: Make this a bit more fancier
	if cloudType == flagset.CloudAws {
		if strings.Contains(reponame, "gitops") {
			if filename == "terraform/base/kubeconfig" {
				log.Debug().Msgf("file not included on commit: '%s'", filename)
				return false
			}
		}

	}
	return true
}

//GitAddWithFilter Check workdir for files to commit
//filter out the undersired ones based on context
func GitAddWithFilter(cloudType string, reponame string, w *git.Worktree) error {
	status, err := w.Status()
	if err != nil {
		log.Debug().Msgf("error getting worktree status: %s", err)
	}

	for file, s := range status {
		log.Printf("the file is %s the status is %v", file, s.Worktree)
		if AppendFile(viper.GetString("cloud"), "gitops", file) {
			_, err = w.Add(file)
			if err != nil {
				log.Error().Err(err).Msgf("error getting worktree status: %s", err)
			}
		}
	}
	return nil
}
