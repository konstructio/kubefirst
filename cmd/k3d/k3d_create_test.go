package k3d

import (
	"os"
	"testing"

	"github.com/kubefirst/kubefirst/internal/logging"
	"github.com/kubefirst/runtime/pkg/k3d"
	"github.com/stretchr/testify/require"
)

func TestPreChecks(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		envs map[string]string
		err  bool
	}{
		{
			name: "invalid git provider - should fail",
			args: []string{"--git-provider=invalid"},
			err:  true,
		},
		{
			name: "github user and github org specified - should fail",
			args: []string{"--github-user=invalid", "--github-org=invalid"},
			err:  true,
		},
		{
			name: "gitlab provider without gitlab group - should fail",
			args: []string{"--git-provider=gitlab"},
			err:  true,
		},
		{
			name: "invalid git protocol - should fail",
			args: []string{"--git-protocol=ftp"},
			err:  true,
		},
		{
			name: "invalid catalog items, - should fail",
			args: []string{"--install-catalog-apps=invalid"},
			envs: map[string]string{
				GITLAB_TOKEN: "abc",
			},
			err: true,
		},
	}

	for _, tc := range testCases {
		cmd := NewK3dCreateCommand()
		cmd.SetArgs(tc.args)

		for k, v := range tc.envs {
			t.Setenv(k, v)
		}

		require.Error(t, cmd.Execute(), tc.name)
	}
}

func TestInstallTools(t *testing.T) {
	o := createOptions{}
	config := k3d.GetConfig(o.clusterName, o.gitProvider, o.gitOwner, o.gitProtocol)

	// required for storing config state
	logging.Init()

	require.NoError(t, o.downloadTools(config))

	require.FileExists(t, config.TerraformClient)
	require.FileExists(t, config.K3dClient)
	require.FileExists(t, config.KubectlClient)
	require.FileExists(t, config.MkCertClient)

	t.Cleanup(func() {
		os.RemoveAll(config.K1Dir)
		os.Remove("~/.kubefirst")
	})
}
