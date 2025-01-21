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
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	pkg "github.com/konstructio/kubefirst-api/pkg/utils"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/konstructio/kubefirst/internal/gitShim"
	"github.com/konstructio/kubefirst/internal/launch"
	"github.com/konstructio/kubefirst/internal/provision"
	"github.com/konstructio/kubefirst/internal/step"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/konstructio/kubefirst/internal/utilities"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type KubefirstAWSClient struct {
	stepper  step.Stepper
	cliFlags types.CliFlags
}

func (c *KubefirstAWSClient) CreateManagementCluster(ctx context.Context, catalogApps []apiTypes.GitopsCatalogApp) error {

	initializeConfigStep := c.stepper.NewStep("Initialize Config")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.cliFlags.CloudRegion))
	if err != nil {
		wrerr := fmt.Errorf("unable to load AWS SDK config: %w", err)
		initializeConfigStep.Complete(wrerr)
		return wrerr
	}

	err = ValidateProvidedFlags(ctx, cfg, c.cliFlags.GitProvider, c.cliFlags.AMIType, c.cliFlags.NodeType)
	if err != nil {
		wrerr := fmt.Errorf("failed to validate provided flags: %w", err)
		initializeConfigStep.Complete(wrerr)
		return wrerr
	}

	utilities.CreateK1ClusterDirectory(c.cliFlags.ClusterName)

	// If cluster setup is complete, return
	clusterSetupComplete := viper.GetBool("kubefirst-checks.cluster-install-complete")
	if clusterSetupComplete {
		err = fmt.Errorf("this cluster install process has already completed successfully, no need to run again")
		return initializeConfigStep.Complete(err)
	}

	creds, err := getSessionCredentials(ctx, cfg.Credentials)
	if err != nil {
		wrerr := fmt.Errorf("failed to retrieve AWS credentials: %w", err)
		initializeConfigStep.Complete(wrerr)
		return wrerr
	}

	viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyID)
	viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
	viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
	if err := viper.WriteConfig(); err != nil {
		wrerr := fmt.Errorf("failed to write config: %w", err)
		initializeConfigStep.Complete(wrerr)
		return wrerr
	}

	initializeConfigStep.Complete(nil)

	validateGitStep := c.stepper.NewStep("Setup Gitops Repository")

	gitAuth, err := gitShim.ValidateGitCredentials(c.cliFlags.GitProvider, c.cliFlags.GithubOrg, c.cliFlags.GitlabGroup)
	if err != nil {
		wrerr := fmt.Errorf("failed to validate Git credentials: %w", err)
		validateGitStep.Complete(wrerr)
		return wrerr
	}

	executionControl := viper.GetBool(fmt.Sprintf("kubefirst-checks.%s-credentials", c.cliFlags.GitProvider))
	if !executionControl {
		newRepositoryNames := []string{"gitops", "metaphor"}
		newTeamNames := []string{"admins", "developers"}

		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  c.cliFlags.GitProvider,
			GitToken:     gitAuth.Token,
			GitOwner:     gitAuth.Owner,
			Repositories: newRepositoryNames,
			Teams:        newTeamNames,
		}

		err = gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			wrerr := fmt.Errorf("failed to initialize Git provider: %w", err)
			validateGitStep.Complete(wrerr)
			return wrerr
		}
	}

	viper.Set(fmt.Sprintf("kubefirst-checks.%s-credentials", c.cliFlags.GitProvider), true)
	if err := viper.WriteConfig(); err != nil {
		wrerr := fmt.Errorf("failed to write config: %w", err)
		validateGitStep.Complete(wrerr)
		return wrerr
	}

	validateGitStep.Complete(nil)
	setupK3dClusterStep := c.stepper.NewStep("Setup k3d Cluster")

	k3dClusterCreationComplete := viper.GetBool("launch.deployed")
	isK1Debug := strings.ToLower(os.Getenv("K1_LOCAL_DEBUG")) == "true"

	if !k3dClusterCreationComplete && !isK1Debug {
		err = launch.Up(nil, true, c.cliFlags.UseTelemetry)

		if err != nil {
			wrerr := fmt.Errorf("failed to setup k3d cluster: %w", err)
			setupK3dClusterStep.Complete(wrerr)
			return wrerr
		}
	}

	err = pkg.IsAppAvailable(fmt.Sprintf("%s/api/proxyHealth", cluster.GetConsoleIngressURL()), "kubefirst api")
	if err != nil {
		wrerr := fmt.Errorf("failed to check kubefirst api health: %w", err)
		setupK3dClusterStep.Complete(wrerr)
		return wrerr
	}

	setupK3dClusterStep.Complete(nil)
	createMgmtClusterStep := c.stepper.NewStep("Create Management Cluster")

	if err := provision.CreateMgmtCluster(gitAuth, c.cliFlags, catalogApps); err != nil {
		wrerr := fmt.Errorf("failed to create management cluster: %w", err)
		createMgmtClusterStep.Complete(wrerr)
		return wrerr
	}

	createMgmtClusterStep.Complete(nil)

	return nil
}

func ValidateProvidedFlags(ctx context.Context, cfg aws.Config, gitProvider, amiType, nodeType string) error {

	// TODO: Handle for non-bubbletea
	// progress.AddStep("Validate provided flags")

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

	ssmClient := ssm.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, &ec2.DescribeInstanceTypesInput{})

	if err := validateAMIType(ctx, amiType, nodeType, ssmClient, ec2Client, paginator); err != nil {
		return fmt.Errorf("failed to validate ami type for node group: %w", err)
	}

	// TODO: Handle for non-bubbletea
	// progress.CompleteStep("Validate provided flags")

	return nil
}

func getSessionCredentials(ctx context.Context, cp aws.CredentialsProvider) (*aws.Credentials, error) {
	// Retrieve credentials
	creds, err := cp.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &creds, nil
}

func validateAMIType(ctx context.Context, amiType, nodeType string, ssmClient ssmClienter, ec2Client ec2Clienter, paginator paginator) error {
	ssmParameterName, ok := supportedAMITypes[amiType]
	if !ok {
		return fmt.Errorf("not a valid ami type: %q", amiType)
	}

	amiID, err := getLatestAMIFromSSM(ctx, ssmClient, ssmParameterName)
	if err != nil {
		return fmt.Errorf("failed to get AMI ID from SSM: %w", err)
	}

	architecture, err := getAMIArchitecture(ctx, ec2Client, amiID)
	if err != nil {
		return fmt.Errorf("failed to get AMI architecture: %w", err)
	}

	instanceTypes, err := getSupportedInstanceTypes(ctx, paginator, architecture)
	if err != nil {
		return fmt.Errorf("failed to get supported instance types: %w", err)
	}

	for _, instanceType := range instanceTypes {
		if instanceType == nodeType {
			return nil
		}
	}

	return fmt.Errorf("node type %q not supported for %q\nSupported instance types: %s", nodeType, amiType, instanceTypes)
}

type ssmClienter interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func getLatestAMIFromSSM(ctx context.Context, ssmClient ssmClienter, parameterName string) (string, error) {
	input := &ssm.GetParameterInput{
		Name: aws.String(parameterName),
	}
	output, err := ssmClient.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failure when fetching parameters: %w", err)
	}

	if output == nil || output.Parameter == nil || output.Parameter.Value == nil {
		return "", fmt.Errorf("invalid parameter value found for %q", parameterName)
	}

	return *output.Parameter.Value, nil
}

type ec2Clienter interface {
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

func getAMIArchitecture(ctx context.Context, ec2Client ec2Clienter, amiID string) (string, error) {
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

type paginator interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}

func getSupportedInstanceTypes(ctx context.Context, p paginator, architecture string) ([]string, error) {
	var instanceTypes []string
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load next pages for instance types: %w", err)
		}

		for _, instanceType := range page.InstanceTypes {
			if slices.Contains(instanceType.ProcessorInfo.SupportedArchitectures, ec2Types.ArchitectureType(architecture)) {
				instanceTypes = append(instanceTypes, string(instanceType.InstanceType))
			}
		}
	}
	return instanceTypes, nil
}
