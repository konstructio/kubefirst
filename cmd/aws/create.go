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
	"github.com/aws/aws-sdk-go-v2/service/sts"
	bl "github.com/charmbracelet/log"
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

	err = ValidateProvidedFlags(cliFlags.GitProvider, cliFlags.AmiType, cliFlags.CloudRegion)
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

	creds, err := ValidateAWSRegionAndRetrieveCredentials(cloudRegionFlag)
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

func ValidateProvidedFlags(gitProvider, amiType, cloudRegion string) error {
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

	if err := ValidateAMIType(amiType, cloudRegion); err != nil {
		progress.Error(err.Error())
		return fmt.Errorf("failed to validte ami type for node group: %w", err)
	}

	progress.CompleteStep("Validate provided flags")

	return nil
}

func ValidateAWSRegionAndRetrieveCredentials(cloudRegion string) (*aws.Credentials, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cloudRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}

	// Validate region by creating a client
	stsClient := sts.NewFromConfig(cfg)
	_, err = stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to validate AWS region: %w", err)
	}

	// Retrieve credentials
	creds, err := cfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &creds, nil
}

func ValidateAMIType(amiType, region string) error {
	ssm_types := map[string]string{
		"AL2_x86_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
		"AL2_ARM_64":                 "/aws/service/eks/optimized-ami/1.29/amazon-linux-2-arm64/recommended/image_id",
		"BOTTLEROCKET_ARM_64":        "/aws/service/bottlerocket/aws-k8s-1.29/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64":        "/aws/service/bottlerocket/aws-k8s-1.29/x86_64/latest/image_id",
		"BOTTLEROCKET_ARM_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/arm64/latest/image_id",
		"BOTTLEROCKET_x86_64_NVIDIA": "/aws/service/bottlerocket/aws-k8s-1.29-nvidia/x86_64/latest/image_id",
	}

	_, ok := ssm_types[amiType]
	if !ok {
		return fmt.Errorf("not a valid ami type")
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %w", err)
	}
	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	ssmParameterName := ssm_types["BOTTLEROCKET_x86_64_NVIDIA"]

	amiID, err := GetLatestAMIFromSSM(ssmClient, ssmParameterName)
	if err != nil {
		return fmt.Errorf("failed to get AMI ID from SSM: %w", err)
	}
	bl.Info("Retrieved AMI ID: %s\n", amiID)

	architecture, err := GetAMIArchitecture(ec2Client, amiID)
	if err != nil {
		return fmt.Errorf("failed to get AMI architecture: %w", err)
	}
	fmt.Printf("AMI Architecture: %s\n", architecture)

	instanceTypes, err := GetSupportedInstanceTypes(ec2Client, architecture)
	if err != nil {
		return fmt.Errorf("failed to get supported instance types: %w", err)
	}

	fmt.Println("Supported Instance Types:")
	for _, instanceType := range instanceTypes {
		fmt.Println(instanceType)
	}

	return nil
}

func GetLatestAMIFromSSM(ssmClient *ssm.Client, parameterName string) (string, error) {

	input := &ssm.GetParameterInput{
		Name: aws.String(parameterName),
	}
	output, err := ssmClient.GetParameter(context.TODO(), input)
	if err != nil {
		return "", err
	}

	return *output.Parameter.Value, nil
}

func GetAMIArchitecture(ec2Client *ec2.Client, amiID string) (string, error) {
	input := &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}
	output, err := ec2Client.DescribeImages(context.TODO(), input)
	if err != nil {
		return "", err
	}

	if len(output.Images) == 0 {
		return "", fmt.Errorf("no images found for AMI ID: %s", amiID)
	}

	val := output.Images[0]
	return string(val.Architecture), nil
}

func GetSupportedInstanceTypes(ec2Client *ec2.Client, architecture string) ([]string, error) {
	input := &ec2.DescribeInstanceTypesInput{}

	var instanceTypes []string
	paginator := ec2.NewDescribeInstanceTypesPaginator(ec2Client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}

		for _, instanceType := range page.InstanceTypes {
			if string(instanceType.ProcessorInfo.SupportedArchitectures[0]) == architecture {
				instanceTypes = append(instanceTypes, string(instanceType.InstanceType))
			}
		}
	}

	return instanceTypes, nil
}
