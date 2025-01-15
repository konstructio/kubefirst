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

func TestCanRoleDoAction(t *testing.T) {
	tests := []struct {
		name           string
		mockFn         func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error)
		roleArn        string
		actions        []string
		expectedResult bool
		expectedError  error
	}{
		{
			name: "all actions allowed",
			mockFn: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
				return &iam.SimulatePrincipalPolicyOutput{
					EvaluationResults: []types.EvaluationResult{
						{
							EvalActionName: aws.String("eks:CreateCluster"),
							EvalDecision:   "allowed",
						},
						{
							EvalActionName: aws.String("s3:PutObject"),
							EvalDecision:   "allowed",
						},
					},
				}, nil
			},
			roleArn:        "arn:aws:iam::123456789012:role/MyRole",
			actions:        []string{"eks:CreateCluster", "s3:PutObject"},
			expectedResult: true,
			expectedError:  nil,
		},
		{
			name: "some actions not allowed",
			mockFn: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
				return &iam.SimulatePrincipalPolicyOutput{
					EvaluationResults: []types.EvaluationResult{
						{
							EvalActionName: aws.String("eks:CreateCluster"),
							EvalDecision:   "allowed",
						},
						{
							EvalActionName: aws.String("s3:PutObject"),
							EvalDecision:   "explicitDeny",
						},
					},
				}, nil
			},
			roleArn:        "arn:aws:iam::123456789012:role/MyRole",
			actions:        []string{"eks:CreateCluster", "s3:PutObject"},
			expectedResult: false,
			expectedError:  nil,
		},
		{
			name: "simulate policy error",
			mockFn: func(ctx context.Context, params *iam.SimulatePrincipalPolicyInput, optFns ...func(*iam.Options)) (*iam.SimulatePrincipalPolicyOutput, error) {
				return nil, errors.New("simulation failed")
			},
			roleArn:        "arn:aws:iam::123456789012:role/MyRole",
			actions:        []string{"eks:CreateCluster"},
			expectedResult: false,
			expectedError:  errors.New("policy simulation error: simulation failed"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockClient := &mockIamClient{
				FnSimulatePrincipalPolicy: test.mockFn,
			}
			checker := &Checker{IAMClient: mockClient}
			result, err := checker.CanRoleDoAction(context.Background(), test.roleArn, test.actions)

			if test.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, test.expectedError.Error())
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expectedResult, result)
		})
	}
}
