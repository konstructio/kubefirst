package helpers

import "github.com/spf13/viper"

// SetCompletionFlags sets specific config flags to mark status of an install
func SetCompletionFlags(cloudProvider string, gitProvider string) {
	viper.Set("kubefirst.cloud-provider", cloudProvider)
	viper.Set("kubefirst.git-provider", gitProvider)
	viper.Set("kubefirst.setup-complete", true)
	viper.WriteConfig()
}

// GetCompletionFlags gets specific config flags to mark status of an install
func GetCompletionFlags() CompletionFlags {
	return CompletionFlags{
		CloudProvider: viper.GetString("kubefirst.cloud-provider"),
		GitProvider:   viper.GetString("kubefirst.git-provider"),
		SetupComplete: viper.GetBool("kubefirst.setup-complete"),
	}
}
