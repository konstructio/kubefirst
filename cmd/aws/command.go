/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/konstructio/kubefirst-api/pkg/constants"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/common"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Create
	alertsEmailFlag          string
	ciFlag                   bool
	cloudRegionFlag          string
	clusterNameFlag          string
	clusterTypeFlag          string
	dnsProviderFlag          string
	githubOrgFlag            string
	gitlabGroupFlag          string
	gitProviderFlag          string
	gitProtocolFlag          string
	gitopsTemplateURLFlag    string
	gitopsTemplateBranchFlag string
	domainNameFlag           string
	subdomainNameFlag        string
	useTelemetryFlag         bool
	ecrFlag                  bool
	nodeTypeFlag             string
	nodeCountFlag            string
	installCatalogApps       string
	installKubefirstProFlag  bool
	amiType                  string

	// Supported argument arrays
	supportedDNSProviders        = []string{"aws", "cloudflare"}
	supportedGitProviders        = []string{"github", "gitlab"}
	supportedGitProtocolOverride = []string{"https", "ssh"}
	supportedAMITypes            = map[string]string{
		"AL2_x86_64":                 "/aws/service/eks/optimized-ami/1.31/amazon-linux-2/recommended/image_id",
		"AL2_ARM_64":                 "/aws/service/eks/optimized-ami/1.31/amazon-linux-2-arm64/recommended/image_id",
		"BOTTLEROCKET_ARM_64":        "/aws/service/bottlerocket/aws-k8s-1.31/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64":        "/aws/service/bottlerocket/aws-k8s-1.31/x86_64/latest/image_id",
		"BOTTLEROCKET_ARM_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.31-nvidia/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.31-nvidia/x86_64/latest/image_id",
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
		RunE: func(cmd *cobra.Command, _ []string) error {
			cloudProvider := "aws"
			estimatedDurationMin := 40
			ctx := cmd.Context()
			stepper := step.NewStepFactory(cmd.ErrOrStderr())

			stepper.DisplayLogHints(cloudProvider, estimatedDurationMin)

			stepper.NewProgressStep("Validate Configuration")

			cliFlags, err := utilities.GetFlags(cmd, cloudProvider)
			if err != nil {
				wrerr := fmt.Errorf("failed to get flags: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			catalogApps, err := catalog.ValidateCatalogApps(ctx, cliFlags.InstallCatalogApps)
			if err != nil {
				wrerr := fmt.Errorf("invalid catalog apps: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cliFlags.CloudRegion))
			if err != nil {
				wrerr := fmt.Errorf("failed to load AWS SDK config: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			err = ValidateProvidedFlags(ctx, cfg, cliFlags.GitProvider, cliFlags.AMIType, cliFlags.NodeType)
			if err != nil {
				wrerr := fmt.Errorf("failed to validate provided flags: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			creds, err := getSessionCredentials(ctx, cfg.Credentials)
			if err != nil {
				wrerr := fmt.Errorf("failed to get session credentials: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
			viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
			viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
			if err := viper.WriteConfig(); err != nil {
				wrerr := fmt.Errorf("failed to write config: %w", err)
				stepper.FailCurrentStep(wrerr)
				return wrerr
			}

			clusterClient := cluster.Client{}

			provision := provision.NewProvisioner(provision.NewProvisionWatcher(cliFlags.ClusterName, &clusterClient), stepper)

			if err := provision.ProvisionManagementCluster(ctx, &cliFlags, catalogApps); err != nil {
				stepper.FailCurrentStep(err)
				return fmt.Errorf("failed to provision aws management cluster: %w", err)
			}

			return nil
		},
	}

	awsDefaults := constants.GetCloudDefaults().Aws

	// todo review defaults and update descriptions
	createCmd.Flags().StringVar(&alertsEmailFlag, "alerts-email", "", "email address for let's encrypt certificate notifications (required)")
	createCmd.MarkFlagRequired("alerts-email")
	createCmd.Flags().BoolVar(&ciFlag, "ci", false, "if running kubefirst in ci, set this flag to disable interactive features")
	createCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")
	createCmd.Flags().StringVar(&clusterNameFlag, "cluster-name", "kubefirst", "the name of the cluster to create")
	createCmd.Flags().StringVar(&clusterTypeFlag, "cluster-type", "mgmt", "the type of cluster to create (i.e. mgmt|workload)")
	createCmd.Flags().StringVar(&nodeCountFlag, "node-count", awsDefaults.NodeCount, "the node count for the cluster")
	createCmd.Flags().StringVar(&nodeTypeFlag, "node-type", awsDefaults.InstanceSize, "the instance size of the cluster to create")
	createCmd.Flags().StringVar(&dnsProviderFlag, "dns-provider", "aws", fmt.Sprintf("the dns provider - one of: %q", supportedDNSProviders))
	createCmd.Flags().StringVar(&subdomainNameFlag, "subdomain", "", "the subdomain to use for DNS records (Cloudflare)")
	createCmd.Flags().StringVar(&domainNameFlag, "domain-name", "", "the Route53/Cloudflare hosted zone name to use for DNS records (i.e. your-domain.com|subdomain.your-domain.com) (required)")
	createCmd.MarkFlagRequired("domain-name")
	createCmd.Flags().StringVar(&gitProviderFlag, "git-provider", "github", fmt.Sprintf("the git provider - one of: %q", supportedGitProviders))
	createCmd.Flags().StringVar(&gitProtocolFlag, "git-protocol", "ssh", fmt.Sprintf("the git protocol - one of: %q", supportedGitProtocolOverride))
	createCmd.Flags().StringVar(&githubOrgFlag, "github-org", "", "the GitHub organization for the new gitops and metaphor repositories - required if using github")
	createCmd.Flags().StringVar(&gitlabGroupFlag, "gitlab-group", "", "the GitLab group for the new gitops and metaphor projects - required if using gitlab")
	createCmd.Flags().StringVar(&gitopsTemplateBranchFlag, "gitops-template-branch", "", "the branch to clone for the gitops-template repository")
	createCmd.Flags().StringVar(&gitopsTemplateURLFlag, "gitops-template-url", "https://github.com/konstructio/gitops-template.git", "the fully qualified url to the gitops-template repository to clone")
	createCmd.Flags().StringVar(&installCatalogApps, "install-catalog-apps", "", "comma separated values to install after provision")
	createCmd.Flags().BoolVar(&useTelemetryFlag, "use-telemetry", true, "whether to emit telemetry")
	createCmd.Flags().BoolVar(&ecrFlag, "ecr", false, "whether or not to use ecr vs the git provider")
	createCmd.Flags().BoolVar(&installKubefirstProFlag, "install-kubefirst-pro", true, "whether or not to install kubefirst pro")
	createCmd.Flags().StringVar(&amiType, "ami-type", "AL2_x86_64", fmt.Sprintf("the ami type for node group - one of: %q", getSupportedAMITypes()))

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

	quotaCmd.Flags().StringVar(&cloudRegionFlag, "cloud-region", "us-east-1", "the aws region to provision infrastructure in")

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
