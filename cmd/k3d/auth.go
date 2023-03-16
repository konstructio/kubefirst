package k3d

import (
	"github.com/kubefirst/kubefirst/internal/helpers"
	"github.com/spf13/cobra"
)

func getK3dAuth(cmd *cobra.Command, args []string) error {
	helpers.ParseAuthData(gitProviderFlag)
	return nil
}
