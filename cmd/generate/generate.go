package generate

import (
	"fmt"
	"os"

	"github.com/kubefirst/runtime/pkg/gitClient"
	"github.com/spf13/cobra"
)

func generate(cmd *cobra.Command, args []string) error {

	fmt.Println("clone git repo")
	const localGitopsPath = "/Users/jared/basura/gitops-templates/"

	gitopsRepo, err := gitClient.CloneWithTokenAuth(os.Getenv("GITHUB_TOKEN"), "main", localGitopsPath, "https://github.com/your-company-io/gitops.git")
	if err != nil {
		fmt.Errorf("error: ", err)
	}
	fmt.Println(gitopsRepo)

	fmt.Println("read secrets from vault")
	fmt.Println("detokenize")

	return nil
}
