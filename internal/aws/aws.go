package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamClienter interface {
	SimulatePrincipalPolicy(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error)
}

// AWSChecker is a struct that holds an IAM client
// to check if a role can perform a given action
type AWSChecker struct {
	iamClient iamClienter
}

// NewChecker creates a new AWSChecker instance
func NewChecker(ctx context.Context) (*AWSChecker, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	iamClient := iam.NewFromConfig(cfg)
	return &AWSChecker{iamClient: iamClient}, nil
}

// CheckIfRoleCan calls the IAM Policy Simulator for a given set of permissions
// For example, to know if a role can create an EKS cluster, you can call this function
// with the role ARN and the action "eks:CreateCluster":
//
//	canCreateCluster, err := CheckIfRoleCan(ctx, "arn:aws:iam::123456789012:role/MyRole", []string{"eks:CreateCluster"})
func (a *AWSChecker) CheckIfRoleCan(ctx context.Context, roleArn string, actions []string) (bool, error) {
	simulateOutput, err := a.iamClient.SimulatePrincipalPolicy(ctx, &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(roleArn),
		ActionNames:     actions,
		ResourceArns:    []string{"*"}, // or a more specific ARN if you want
	})
	if err != nil {
		return false, fmt.Errorf("policy simulation error: %w", err)
	}

	// Evaluate simulation results
	for _, res := range simulateOutput.EvaluationResults {
		if res.EvalActionName != nil && *res.EvalActionName == "eks:CreateCluster" {
			// res.EvalDecision is one of: allowed / explicitDeny / implicitDeny
			if res.EvalDecision == "allowed" {
				return true, nil
			}
		}
	}

	return false, nil
}
