/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/catalog"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func createAws(cmd *cobra.Command, _ []string) error {
	cliFlags, err := utilities.GetFlags(cmd, "aws")
	if err != nil {
		progress.Error(err.Error())
		return nil
	}

	progress.DisplayLogHints(40)

	isValid, catalogApps, err := catalog.ValidateCatalogApps(cliFlags.InstallCatalogApps)
	if !isValid {
		return fmt.Errorf("invalid catalog apps: %w", err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cliFlags.CloudRegion))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	ctx := context.Background()

	err = ValidateProvidedFlags(ctx, cfg, cliFlags.GitProvider, cliFlags.AMIType, cliFlags.NodeType)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate provided flags: %w", err)
	}

	utilities.CreateK1ClusterDirectory(cliFlags.ClusterName)

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err = fmt.Errorf("this cluster install process has already completed successfully")
		progress.Error(err.Error())
		return nil
	}

	creds, err := getSessionCredentials(ctx, cfg.Credentials)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
	viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	gitAuth, err := gitShim.ValidateGitCredentials(cliFlags.GitProvider, cliFlags.GithubOrg, cliFlags.GitlabGroup)
	if err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validate Git credentials: %w", err)
	}

	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider))
	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  cliFlags.GitProvider,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("failed to initialize Git provider: %w", err)
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", cliFlags.GitProvider), true)
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		launch.Up(nil, true, cliFlags.UseTelemetry)
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		progress.Error("unable to start kubefirst api")
		return fmt.Errorf("failed to check kubefirst API availability: %w", err)
	}

	if err := provision.CreateMgmtCluster(gitAuth, cliFlags, catalogApps); err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to create management cluster: %w", err)
	}

	return nil
}

func ValidateProvidedFlags(ctx context.Context, cfg aws.Config, gitProvider, amiType, nodeType string) error {
	progress.AddStep("Validate provided flags")

	// Validate required environment variables for dns provider
	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch gitProvider {
	case "github":
		key, err := internalssh.GetHostKey("github.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan github.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "github.com", key.Type())
	case "gitlab":
		key, err := internalssh.GetHostKey("gitlab.com")
		if err != nil {
			return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan gitlab.com >> ~/.ssh/known_hosts` to remedy: %w", err)
		}
		log.Info().Msgf("%q %s", "gitlab.com", key.Type())
	}

	if err := ValidateAMIType(ctx, cfg, amiType, nodeType); err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validte ami type for node group: %w", err)
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}

func getSessionCredentials(ctx context.Context, cfg aws.CredentialsProvider) (*aws.Credentials, error) {

	// Retrieve credentials
	creds, err := cfg.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &creds, nil
}

func ValidateAMIType(ctx context.Context, cfg aws.Config, amiType, nodeType string) error {
	ssmTypes := map[string]string{
		"AL2_x86_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
		"AL2_ARM_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2-arm64/recommended/image_id",
		"BOTTLEROCKET_ARM_64":        "/aws/service/bottlerocket/aws-k8s-1.29/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64":        "/aws/service/bottlerocket/aws-k8s-1.29/x86_64/latest/image_id",
		"BOTTLEROCKET_ARM_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/x86_64/latest/image_id",
	}

	_, ok := ssmTypes[amiType]
	if !ok {
		return fmt.Errorf("not a valid ami type")
	}

	ssmClient := ssm.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	ssmParameterName := ssmTypes[amiType]

	amiID, err := GetLatestAMIFromSSM(ctx, ssmClient, ssmParameterName)
	if err != nil {
		return fmt.Errorf("failed to get AMI ID from SSM: %w", err)
	}
	log.Info().Msgf("ami type is  %s", amiType)

	architecture, err := GetAMIArchitecture(ctx, ec2Client, amiID)
	if err != nil {
		return fmt.Errorf("failed to get AMI architecture: %w", err)
	}

	instanceTypes, err := GetSupportedInstanceTypes(cfg, string(architecture))
	if err != nil {
		return fmt.Errorf("failed to get supported instance types: %w", err)
	}

	for _, instanceType := range instanceTypes {
		if instanceType == nodeType {
			return nil
		}
	}

	return fmt.Errorf("node type %s not supported for %s", nodeType, amiType)
}

type ssmClienter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func GetLatestAMIFromSSM(ctx context.Context, ssmClient ssmClienter, parameterName string) (string, error) {
	input := &ssm.GetParameterInput{
		Name: aws.String(parameterName),
	}
	output, err := ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to initialise ssm client: %w", err)
	}

	return *output.Parameter.Value, nil
}

type ec2Clienter interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

func GetAMIArchitecture(ctx context.Context, ec2Client ec2Clienter, amiID string) (string, error) {
	input := &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}
	output, err := ec2Client.DescribeImages(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to describe images: %w", err)
	}

	if len(output.Images) == 0 {
		return "", fmt.Errorf("no images found for AMI ID: %s", amiID)
	}

	return string(output.Images[0].Architecture), nil
}

func GetSupportedInstanceTypes(cfg aws.Config, architecture string) ([]string, error) {
	ec2Client := ec2.NewFromConfig(cfg)
	input := &ec2.DescribeInstanceTypesInput{}

	var instanceTypes []string
	paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to load next pages for instance types: %w", err)
		}

		for _, instanceType := range page.InstanceTypes {
			if string(instanceType.ProcessorInfo.SupportedArchitectures[0]) == architecture {
				instanceTypes = append(instanceTypes, string(instanceType.InstanceType))
			}
		}
	}

	return instanceTypes, nil
}
