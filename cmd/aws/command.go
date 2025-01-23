/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"

	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/spf13/cobra"
)

var (
	// Supported argument arrays
	supportedDNSProviders        = []string{"aws", "cloudflare"}
	supportedGitProviders        = []string{"github", "gitlab"}
	supportedGitProtocolOverride = []string{"https", "ssh"}
	supportedAMITypes            = map[string]string{
		"AL2_x86_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
		"AL2_ARM_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2-arm64/recommended/image_id",
		"BOTTLEROCKET_ARM_64":        "/aws/service/bottlerocket/aws-k8s-1.29/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64":        "/aws/service/bottlerocket/aws-k8s-1.29/x86_64/latest/image_id",
		"BOTTLEROCKET_ARM_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/x86_64/latest/image_id",
	}
)

func NewCommand() *cobra.Command {
	awsCmd := &cobra.Command{
		Use:   "aws",
		Short: "kubefirst aws installation",
		Long:  "kubefirst aws",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println("To learn more about aws in kubefirst, run:")
			fmt.Println("  kubefirst help")

			if progress.Progress != nil {
				progress.Progress.Quit()
			}
		},
	}

	// wire up new commands
	awsCmd.AddCommand(Create(), Destroy(), Quota(), RootCredentials())
	return awsCmd
}

func Create() *cobra.Command {
	createCmd := &cobra.Command{
		Use:              "create",
		Short:            "create the kubefirst platform running in aws",
		TraverseChildren: true,
		RunE:             createAws,
		// PreRun:           common.CheckDocker,
	}

	awsDefaults := constants.GetCloudDefaults().Aws

	// todo review defaults and update descriptions
	createCmd.Flags().String("alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().Bool("ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().String("cloud-region", "us-east-1", "the aws region to provision infrastructure in")
	createCmd.Flags().String("cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().String("cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().String("node-count", awsDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().String("node-type", awsDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().String("dns-provider", "aws", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().String("subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().String("domain-name", "", "the Route53/Cloudflare hosted zone name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().String("git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().String("git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().String("github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().String("gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().String("gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().String("gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().String("install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().Bool("use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().Bool("ecr", false, "whether or not to use ecr vs the git provider")
	createCmd.Flags().Bool("install-kubefirst-pro", true, "whether or not to install kubefirst pro")
	createCmd.Flags().String("ami-type", "AL2_x86_64", fmt.Sprintf("the ami type for node group - one of: %q", getSupportedAMITypes()))
	createCmd.Flags().String("kubernetes-admin-role-arn", "", "the role arn with Kubernetes Admin privileges to use for the creation flow")

	return createCmd
}

func getSupportedAMITypes() []string {
	amiTypes := make([]string, 0, len(supportedAMITypes))
	for k := range supportedAMITypes {
		amiTypes = append(amiTypes, k)
	}
	return amiTypes
}

func Destroy() *cobra.Command {
	destroyCmd := &cobra.Command{
		Use:   "destroy",
		Short: "destroy the kubefirst platform",
		Long:  "deletes the GitHub resources, aws resources, and local content to re-provision",
		RunE:  common.Destroy,
		// PreRun: common.CheckDocker,
	}

	return destroyCmd
}

func Quota() *cobra.Command {
	quotaCmd := &cobra.Command{
		Use:   "quota",
		Short: "Check aws quota status",
		Long:  "Check aws quota status. By default, only ones close to limits will be shown.",
		RunE:  evalAwsQuota,
	}

	quotaCmd.Flags().String("cloud-region", "us-east-1", "the aws region to provision infrastructure in")

	return quotaCmd
}

func RootCredentials() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "root-credentials",
		Short: "retrieve root authentication information for platform components",
		Long:  "retrieve root authentication information for platform components",
		RunE:  common.GetRootCredentials,
	}

	return authCmd
}
