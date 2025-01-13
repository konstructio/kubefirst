package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/stretchr/testify/require"
)

type mockIamClient struct {
	FnSimulatePrincipalPolicy func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error)
}

func (m *mockIamClient) SimulatePrincipalPolicy(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
	if m.FnSimulatePrincipalPolicy != nil {
		return m.FnSimulatePrincipalPolicy(ctx, params, optFns...)
	}
	return nil, errors.New("not implemented")
}

func TestCheckIfRoleCan(t *testing.T) {
	ctx := context.Background()
	roleArn := "arn:aws:iam::123456789012:role/MyRole"
	actions := []string{"eks:CreateCluster"}

	tests := []struct {
		name           string
		mockIamClient  *mockIamClient
		expectedResult bool
		wantErr        bool
	}{
		{
			name: "successful permission check",
			mockIamClient: &mockIamClient{
				FnSimulatePrincipalPolicy: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
					return &iam.SimulatePrincipalPolicyOutput{
						EvaluationResults: []types.EvaluationResult{
							{
								EvalActionName: aws.String("eks:CreateCluster"),
								EvalDecision:   types.PolicyEvaluationDecisionTypeAllowed,
							},
						},
					}, nil
				},
			},
			expectedResult: true,
		},
		{
			name: "permission denied",
			mockIamClient: &mockIamClient{
				FnSimulatePrincipalPolicy: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
					return &iam.SimulatePrincipalPolicyOutput{
						EvaluationResults: []types.EvaluationResult{
							{
								EvalActionName: aws.String("eks:CreateCluster"),
								EvalDecision:   types.PolicyEvaluationDecisionTypeExplicitDeny,
							},
						},
					}, nil
				},
			},
			expectedResult: false,
		},
		{
			name: "error from SimulatePrincipalPolicy",
			mockIamClient: &mockIamClient{
				FnSimulatePrincipalPolicy: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
					return nil, errors.New("simulate policy error")
				},
			},
			expectedResult: false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &AWSChecker{IAMClient: tt.mockIamClient}
			result, err := checker.CheckIfRoleCan(ctx, roleArn, actions)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
