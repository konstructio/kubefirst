/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	internalssh "github.com/konstructio/kubefirst-api/pkg/ssh"
	internalaws "github.com/konstructio/kubefirst/internal/aws"
	"github.com/konstructio/kubefirst/internal/progress"
	"github.com/konstructio/kubefirst/internal/types"
	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
)

const (
	maxRetries = 20
	baseDelay  = 2 * time.Second
)

func ValidateProvidedFlags(ctx context.Context, cfg aws.Config, cliFlags *types.CliFlags) error {
	// Validate required environment variables for dns provider

	stsClient := sts.NewFromConfig(cfg)
	iamClient := iam.NewFromConfig(cfg)
	checker, err := internalaws.NewChecker(ctx)
	if err != nil {
		return fmt.Errorf("failed to perform aws checks: %w", err)
	}

	accessKeyID := viper.Get("kubefirst.state-store-creds.access-key-id")
	secretAccessKeyID := viper.Get("kubefirst.state-store-creds.secret-access-key-id")
	sessionToken := viper.Get("kubefirst.state-store-creds.token")

	if accessKeyID == nil || secretAccessKeyID == nil && sessionToken == nil {
		creds, err := convertLocalCredsToSession(ctx, stsClient, iamClient, checker, cliFlags.KubeAdminRoleARN, cliFlags.ClusterName)
		if err != nil {
			progress.Error(err.Error())
			return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
		}

		viper.Set("kubefirst.state-store-creds.access-key-id", creds.AccessKeyId)
		viper.Set("kubefirst.state-store-creds.secret-access-key-id", creds.SecretAccessKey)
		viper.Set("kubefirst.state-store-creds.token", creds.SessionToken)
		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	if dnsProviderFlag == "cloudflare" {
		if os.Getenv("CF_API_TOKEN") == "" {
			return fmt.Errorf("your CF_API_TOKEN environment variable is not set. Please set and try again")
		}
	}

	switch cliFlags.GitProvider {
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

	if err := validateAMIType(ctx, amiType, cliFlags.NodeType, ssmClient, ec2Client, paginator); err != nil {
		return fmt.Errorf("failed to validate ami type for node group: %w", err)
	}

	return nil
}

const (
	sessionDuration = int32(43200) // We want at least 6 hours (21,600 seconds)
)

var wantedPermissions = []string{
	"eks:*",
	"ec2:*",
	"s3:*",
	"iam:*",
	"dynamodb:*",
	"kms:*",
	"logs:*",
	"application-autoscaling:*",
}

type stsClienter interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

// iamClienter is an interface for IAM operations (CreateRole, AttachRolePolicy, etc.).
type iamClienter interface {
	CreatePolicy(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	AttachRolePolicy(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	GetPolicy(ctx context.Context, params *iam.GetPolicyInput, optFns ...func(*iam.Options)) (*iam.GetPolicyOutput, error)
}

type checkerClienter interface {
	CanRoleDoAction(ctx context.Context, roleArn string, actions []string) (bool, error)
}

func convertLocalCredsToSession(ctx context.Context, stsClient stsClienter, iamClient iamClienter, checker checkerClienter, roleArn, clusterName string) (*awsTypes.Credentials, error) {
	// Check who we are currently (to ensure you're properly authenticated)
	callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}

	// If ARN is empty, create a new role with kubernetes admin permissions
	if roleArn == "" {
		createdArn, err := createKubernetesAdminRole(ctx, clusterName, iamClient, checker, callerIdentity)
		if err != nil {
			return nil, fmt.Errorf("failed to create a new role for EKS clusters: %w", err)
		}
		roleArn = createdArn
	}

	// Check if the currently provided role can perform EKS cluster creation
	// with all the sub-requirements to actually make a cluster.
	canCreateCluster, err := checker.CanRoleDoAction(ctx, roleArn, wantedPermissions)
	if err != nil {
		return nil, fmt.Errorf("failed to check if role %q can create EKS cluster: %w", roleArn, err)
	}
	if !canCreateCluster {
		return nil, fmt.Errorf("role %q does not have permission to create EKS clusters; required permissions: %s", roleArn, wantedPermissions)
	}

	// Check if the currently provided role can perform EKS cluster creation
	// with all the sub-requirements to actually make a cluster.
	canCreateCluster, err = checker.CanRoleDoAction(ctx, *callerIdentity.Arn, []string{"iam:AssumeRole"})
	if err != nil {
		return nil, fmt.Errorf("failed to check if user %q can assume role: %w", roleArn, err)
	}
	if !canCreateCluster {
		return nil, fmt.Errorf("user %q does not have permission to assume roles", roleArn)
	}

	// Create a session name (some unique identifier)
	sessionName := fmt.Sprintf("kubefirst-session-%s", *callerIdentity.UserId)

	creds, err := assumeRoleWithRetry(ctx, stsClient, roleArn, sessionName)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role %s: %w", roleArn, err)
	}

	// // Return the credentials
	return creds, nil
}

// AdditionalRolePolicies is a slice of policy ARNs you want to attach
// to every new "Kubernetes Admin" role created by the function below.
var AdditionalRolePolicies = []string{
	"arn:aws:iam::aws:policy/AmazonEKSServicePolicy",
	"arn:aws:iam::aws:policy/AmazonEKSVPCResourceController",
	// Put your extra policies here, e.g.:
	// "arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess",
}

func createKubernetesAdminRole(ctx context.Context, clusterName string, iamClient iamClienter, checker checkerClienter, callerIdentity *sts.GetCallerIdentityOutput) (string, error) {
	if callerIdentity.Arn == nil {
		return "", errors.New("caller identity ARN is nil")
	}

	// Verify that the current caller has permission to create IAM roles
	wantedRolePermissions := []string{"iam:CreateRole", "iam:AssumeRole", "iam:AttachRolePolicy"}
	canPerformActions, err := checker.CanRoleDoAction(ctx, aws.ToString(callerIdentity.Arn), wantedRolePermissions)
	if err != nil {
		return "", fmt.Errorf("failed to check permission to create a new role: %w", err)
	}
	if !canPerformActions {
		return "", fmt.Errorf("caller %q does not have the required permissions: %s", aws.ToString(callerIdentity.Arn), wantedRolePermissions)
	}

	// Build a custom policy that allows EKS operations
	permissionPolicy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{{
			"Effect":   "Allow",
			"Action":   wantedPermissions,
			"Resource": "*",
		}},
	}

	// Encode the policy as JSON
	permissionPolicyBytes, err := json.Marshal(permissionPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal permission policy JSON: %w", err)
	}

	// Policy name
	policyName := fmt.Sprintf("KubefirstKubernetesAdminPolicy-%s", clusterName)

	// Check if the IAM policy exists
	cp, err := iamClient.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: aws.String(fmt.Sprintf("arn:aws:iam::%s:policy/%s", *callerIdentity.Account, policyName))})
	if err != nil {
		var newError *awshttp.ResponseError
		if errors.As(err, &newError) && newError.HTTPStatusCode() != http.StatusNotFound {
			return "", fmt.Errorf("failed to get policy %q: %w", policyName, err)
		}
	}

	if err == nil && cp.Policy != nil {
		return "", fmt.Errorf("policy %q already exists: please delete the policy and try again", policyName)
	}

	// Create the policy in IAM
	cpo, err := iamClient.CreatePolicy(ctx, &iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(string(permissionPolicyBytes)),
		Description:    aws.String("Policy that allows creating EKS clusters"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create policy %q: %w", policyName, err)
	}

	// Build a trust policy that allows the current caller to assume the new role.
	trustPolicy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{{
			"Effect": "Allow",
			"Principal": map[string]interface{}{
				// Instead of callerIdentity.Arn, we could do:
				//   "AWS": fmt.Sprintf("arn:aws:iam::%s:root", accountId)
				// to allow any principal in your account to assume.
				"AWS": aws.ToString(callerIdentity.Arn),
			},
			"Action": "sts:AssumeRole",
		}},
	}

	// Encode the trust policy as JSON
	trustPolicyBytes, err := json.Marshal(trustPolicy)
	if err != nil {
		return "", fmt.Errorf("failed to marshal trust policy JSON: %w", err)
	}

	// Check if the IAM role exists
	roleName := fmt.Sprintf("KubefirstKubernetesAdminRole-%s", clusterName)

	// Check if a role with this name already exists
	role, err := iamClient.GetRole(ctx, &iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		var newError *awshttp.ResponseError
		if errors.As(err, &newError) && newError.HTTPStatusCode() != http.StatusNotFound {
			return "", fmt.Errorf("failed to get role %q: %w", roleName, err)
		}
	}

	if err == nil && role.Role != nil {
		return "", fmt.Errorf("role %q already exists: please delete the role and try again", roleName)
	}

	// Create the IAM role
	createOut, err := iamClient.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(string(trustPolicyBytes)),
		MaxSessionDuration:       aws.Int32(sessionDuration),
		Description:              aws.String("Role that can create EKS clusters"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create role %q: %w", roleName, err)
	}

	// Attach a policy that lets this role create EKS clusters.
	// The AWS-managed policy "AmazonEKSClusterPolicy" covers cluster creation & management.
	// Real usage often requires more policies for VPC, node groups, etc.
	_, err = iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  createOut.Role.RoleName,
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach AmazonEKSClusterPolicy to role %q: %w", roleName, err)
	}

	// Attach a custom policy that allows EKS operations
	_, err = iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  createOut.Role.RoleName,
		PolicyArn: cpo.Policy.Arn,
	})
	if err != nil {
		return "", fmt.Errorf("failed to attach custom policy %q to role %q: %w", policyName, roleName, err)
	}

	// Attach any additional role policies from the package-level slice
	for _, policyArn := range AdditionalRolePolicies {
		_, err := iamClient.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
			RoleName:  createOut.Role.RoleName,
			PolicyArn: aws.String(policyArn),
		})
		if err != nil {
			return "", fmt.Errorf("failed to attach policy %q to role %q: %w", policyArn, roleName, err)
		}
	}

	// Return the new role ARN
	return aws.ToString(createOut.Role.Arn), nil
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

func assumeRoleWithRetry(ctx context.Context, stsClient stsClienter, roleArn, sessionName string) (*awsTypes.Credentials, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		output, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(roleArn),
			RoleSessionName: aws.String(sessionName),
			DurationSeconds: aws.Int32(sessionDuration),
		})
		if err == nil {
			return output.Credentials, nil
		}

		delay := time.Duration(attempt*attempt) * baseDelay
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("failed to assume role")
}
