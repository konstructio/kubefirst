package helpers

import "github.com/spf13/viper"

// SetCompletionFlags sets specific config flags to mark status of an install
func SetCompletionFlags(cloudProvider string, gitProvider string) {
	viper.Set("kubefirst.cloud-provider", cloudProvider)
	viper.Set("kubefirst.git-provider", gitProvider)
	viper.Set("kubefirst.setup-complete", true)
	viper.WriteConfig()
}
