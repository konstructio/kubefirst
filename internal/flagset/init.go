package flagset

import "github.com/spf13/cobra"

// InitFlags - Load Init Flags and validate values
func InitFlags(cmd *cobra.Command) (GlobalFlags, GithubAddCmdFlags, InstallerGenericFlags, AwsFlags, error) {
	//Please don't change the order of this block, wihtout updating
	// internal/flagset/init_test.go
	globalFlags, err := ProcessGlobalFlags(cmd)
	if err != nil {
		return GlobalFlags{}, GithubAddCmdFlags{}, InstallerGenericFlags{}, AwsFlags{}, err
	}

	githubFlags, err := ProcessGithubAddCmdFlags(cmd)
	if err != nil {
		return GlobalFlags{}, GithubAddCmdFlags{}, InstallerGenericFlags{}, AwsFlags{}, err
	}

	installerFlags, err := ProcessInstallerGenericFlags(cmd)
	if err != nil {
		return GlobalFlags{}, GithubAddCmdFlags{}, InstallerGenericFlags{}, AwsFlags{}, err
	}

	awsFlags, err := ProcessAwsFlags(cmd)
	if err != nil {
		return GlobalFlags{}, GithubAddCmdFlags{}, InstallerGenericFlags{}, AwsFlags{}, err
	}

	//Please don't change the order of this block, wihtout updating
	// internal/flagset/init_test.go
	return globalFlags, githubFlags, installerFlags, awsFlags, nil
}
