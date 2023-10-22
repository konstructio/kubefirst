package k3d

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func mockCommandComplete() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("ci", false, "ci flag")
	cmd.Flags().String("cluster-name", "default", "cluster-name flag")
	cmd.Flags().String("cluster-type", "default", "cluster-type flag")
	cmd.Flags().String("github-org", "", "github-org flag")
	cmd.Flags().String("github-user", "", "github-user flag")
	cmd.Flags().String("gitlab-group", "", "gitlab-group flag")
	cmd.Flags().String("git-provider", "default", "git-provider flag")
	cmd.Flags().String("git-protocol", "default", "git-protocol flag")
	cmd.Flags().String("gitops-template-url", "default", "gitops-template-url flag")
	cmd.Flags().String("gitops-template-branch", "default", "gitops-template-branch flag")
	cmd.Flags().Bool("use-telemetry", false, "use-telemetry flag")
	return cmd
}

func mockCommandIncomplete() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("ci", false, "ci flag")
	return cmd
}

func mockCheckForExistingPortForwards(ports ...int) error {
	return errors.New("port 8080 is in use")
}

func TestRunK3dShouldReturnErrorIfSomeFlagIsNotPresent(t *testing.T) {
	cmd := mockCommandIncomplete()
	args := []string{"create"}
	err := runK3d(cmd, args)

	errorExpected := "flag accessed but not defined: cluster-name"
	if errorExpected != err.Error() {
		t.Errorf("runK3d(%q) returned an error: %v", args, err)
	}
}

func TestRunK3dShouldReturnErrorIfSomeFlagAreNotValid(t *testing.T) {
	cmd := mockCommandComplete()
	cmd.Flags().Set("github-user", "usertest")
	cmd.Flags().Set("github-org", "orgtest")

	args := []string{"create"}
	err := runK3d(cmd, args)

	errorExpected := "only one of --github-user or --github-org can be supplied"
	if errorExpected != err.Error() {
		t.Errorf("runK3d(%q) returned an error: %v", args, err)
	}
}

func TestShouldReturnErrorWhenSomePortsAreNotOpen(t *testing.T) {
	cmd := mockCommandComplete()
	cmd.Flags().Set("github-user", "usertest")

	originalCheckPortsFn := CheckPortsFn
	defer func() { CheckPortsFn = originalCheckPortsFn }()

	CheckPortsFn = mockCheckForExistingPortForwards

	args := []string{"create"}
	err := runK3d(cmd, args)

	errorExpected := "port 8080 is in use - this port is required to set up your kubefirst environment - please close any existing port forwards before continuing"
	if errorExpected != err.Error() {
		t.Errorf("runK3d(%q) returned an error: %v", args, err)
	}
}
