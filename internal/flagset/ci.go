package flagset

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CIFlags - Global flags
type CIFlags struct {
	BranchCI             string
	DestroyBucket        bool
	CIClusterName        string
	CIS3Suffix           string
	CIHostedZoneName     string
	CIFlavor             string
	CIGithubUser         string
	CIGithubOrganization string
}

// DefineCIFlags - Define global flags
func DefineCIFlags(currentCommand *cobra.Command) {
	currentCommand.Flags().String("ci-branch", "", "version/branch used on git clone for ci setup instruction")
	currentCommand.Flags().Bool("destroy-bucket", false, "destroy bucket that stores tfstate of CI infra as code")
	currentCommand.Flags().String("ci-cluster-name", "", "the ci cluster name, used to identify resources on cloud provider")
	currentCommand.Flags().String("ci-s3-suffix", "", "unique identifier for s3 buckets")
	currentCommand.Flags().String("ci-hosted-zone-name", "", "the ci domain to provision the kubefirst platform in")
	currentCommand.Flags().String("ci-flavor", "", "inform which flavor will be installed")
	currentCommand.Flags().String("ci-github-user", "", "inform which github user will be used")
	currentCommand.Flags().String("ci-github-organization", "", "inform which github organization will be used")
}

// ProcessCIFlags - process global flags shared between commands like silent, dry-run and use-telemetry
func ProcessCIFlags(cmd *cobra.Command) (CIFlags, error) {
	flags := CIFlags{}

	branchCI, err := ReadConfigString(cmd, "ci-branch")
	if err != nil {
		log.Printf("Error Processing - ci-branch flag, error: %v", err)
		return flags, err
	}
	flags.BranchCI = branchCI
	viper.Set("ci.branch", branchCI)

	destroyBucket, err := ReadConfigBool(cmd, "destroy-bucket")
	if err != nil {
		log.Printf("Error Processing - destroy-bucket flag, error: %v", err)
		return flags, err
	}
	flags.DestroyBucket = destroyBucket
	viper.Set("destroy.bucket", destroyBucket)

	ciClusterName, err := ReadConfigString(cmd, "ci-cluster-name")
	if err != nil {
		log.Printf("Error Processing - ci-cluster-name flag, error: %v", err)
		return flags, err
	}
	flags.CIClusterName = ciClusterName
	viper.Set("ci.cluster.name", ciClusterName)

	ciS3Suffix, err := ReadConfigString(cmd, "ci-s3-suffix")
	if err != nil {
		log.Printf("Error Processing - ci-s3-suffix flag, error: %v", err)
		return flags, err
	}
	flags.CIS3Suffix = ciS3Suffix
	viper.Set("ci.s3.suffix", ciS3Suffix)

	ciHostedZoneName, err := ReadConfigString(cmd, "ci-hosted-zone-name")
	if err != nil {
		log.Printf("Error Processing - ci-hosted-zone-name flag, error: %v", err)
		return flags, err
	}
	flags.CIHostedZoneName = ciHostedZoneName
	viper.Set("ci.hosted.zone.name", ciHostedZoneName)

	ciFlavor, err := ReadConfigString(cmd, "ci-flavor")
	if err != nil {
		log.Printf("Error Processing - ci-flavor flag, error: %v", err)
		return flags, err
	}
	flags.CIFlavor = ciFlavor
	viper.Set("ci.flavor", ciFlavor)

	ciGithubUser, err := ReadConfigString(cmd, "ci-github-user")
	if err != nil {
		log.Printf("Error Processing - ci-github-user flag, error: %v", err)
		return flags, err
	}
	flags.CIGithubUser = ciGithubUser
	viper.Set("ci.github.user", ciGithubUser)

	ciGithubOrganization, err := ReadConfigString(cmd, "ci-github-organization")
	if err != nil {
		log.Printf("Error Processing - ci-github-organization flag, error: %v", err)
		return flags, err
	}
	flags.CIGithubOrganization = ciGithubOrganization
	viper.Set("ci.github.organization", ciGithubOrganization)

	return flags, nil

}
