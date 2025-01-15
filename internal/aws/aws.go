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

// Checker is a struct that holds an IAM client
// to check if a role can perform a given action
type Checker struct {
	IAMClient iamClienter
}

// NewChecker creates a new AWSChecker instance
func NewChecker(ctx context.Context) (*Checker, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	iamClient := iam.NewFromConfig(cfg)
	return &Checker{IAMClient: iamClient}, nil
}

// CanRoleDoAction calls the IAM Policy Simulator for a given set of permissions
// For example, to know if a role can create an EKS cluster, you can call this function
// with the role ARN and the action "eks:CreateCluster":
//
//	canCreateCluster, err := CanRoleDoAction(ctx, "arn:aws:iam::123456789012:role/MyRole", []string{"eks:CreateCluster"})
func (a *Checker) CanRoleDoAction(ctx context.Context, roleArn string, actions []string) (bool, error) {
	simulateOutput, err := a.IAMClient.SimulatePrincipalPolicy(ctx, &iam.SimulatePrincipalPolicyInput{
		PolicySourceArn: aws.String(roleArn),
		ActionNames:     actions,
		ResourceArns:    []string{"*"}, // or a more specific ARN if you want
	})
	if err != nil {
		return false, fmt.Errorf("policy simulation error: %w", err)
	}

	// Track allowed actions
	allowedActions := make(map[string]bool)

	// Evaluate simulation results for each action
	for _, res := range simulateOutput.EvaluationResults {
		if res.EvalActionName != nil && res.EvalDecision == "allowed" {
			allowedActions[*res.EvalActionName] = true
		}
	}

	// Ensure all actions are allowed
	for _, action := range actions {
		if !allowedActions[action] {
			return false, nil
		}
	}

	return false, nil
}
