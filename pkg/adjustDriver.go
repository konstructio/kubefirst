package pkg

import (
	"fmt"
	"log"
	"os"
	"strings"

	cp "github.com/otiai10/copy"
)

func AdjustGitopsTemplateContent(cloudProvider, gitopsRepoPath, gitProvider string) error {
	os.RemoveAll(gitopsRepoPath + "/components")
	os.RemoveAll(gitopsRepoPath + "/localhost")
	os.RemoveAll(gitopsRepoPath + "/registry")
	os.RemoveAll(gitopsRepoPath + "/validation")
	os.RemoveAll(gitopsRepoPath + "/terraform")
	os.RemoveAll(gitopsRepoPath + "/.gitignore")
	os.RemoveAll(gitopsRepoPath + "/LICENSE")
	os.RemoveAll(gitopsRepoPath + "/README.md")
	os.RemoveAll(gitopsRepoPath + "/atlantis.yaml")
	os.RemoveAll(gitopsRepoPath + "/logo.png")

	driverContent := fmt.Sprintf("%s/%s-%s", gitopsRepoPath, cloudProvider, gitProvider)
	opt := cp.Options{
		Skip: func(src string) (bool, error) {
			if strings.HasSuffix(src, ".git") {
				return true, nil
			} else if strings.Index(src, "/.terraform") > 0 {
				return true, nil
			}
			//Add more stuff to be ignored here
			return false, nil

		},
	}
	err := cp.Copy(driverContent, gitopsRepoPath, opt)
	if err != nil {
		log.Println("Error populating gitops with local setup:", err)
		return err
	}
	os.RemoveAll(driverContent)
	return nil
}
