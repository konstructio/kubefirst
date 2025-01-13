package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/stretchr/testify/require"
)

type mockStsClient struct {
	FnGetCallerIdentity func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
	FnAssumeRole        func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
}

func (m *mockStsClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	if m.FnGetCallerIdentity != nil {
		return m.FnGetCallerIdentity(ctx, params, optFns...)
	}

	return nil, errors.New("not implemented")
}

func (m *mockStsClient) AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
	if m.FnAssumeRole != nil {
		return m.FnAssumeRole(ctx, params, optFns...)
	}

	return nil, errors.New("not implemented")
}

func TestValidateCredentials(t *testing.T) {
	ctx := context.Background()
	roleArn := "arn:aws:iam::123456789012:role/example-role"

	tests := []struct {
		name              string
		mockStsClient     *mockStsClient
		wantErr           bool
		expectedUserId    string
		expectedSessionId string
	}{
		{
			name: "successful conversion",
			mockStsClient: &mockStsClient{
				FnGetCallerIdentity: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return &sts.GetCallerIdentityOutput{
						UserId: aws.String("user-123"),
					}, nil
				},
				FnAssumeRole: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					return &sts.AssumeRoleOutput{
						Credentials: &types.Credentials{
							AccessKeyId:     aws.String("access-key-id"),
							SecretAccessKey: aws.String("secret-access-key"),
							SessionToken:    aws.String("session-token"),
						},
					}, nil
				},
			},
			expectedUserId:    "user-123",
			expectedSessionId: "kubefirst-session-user-123",
		},
		{
			name: "failed to get caller identity",
			mockStsClient: &mockStsClient{
				FnGetCallerIdentity: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return nil, errors.New("failed to get caller identity")
				},
			},
			wantErr: true,
		},
		{
			name: "failed to assume role",
			mockStsClient: &mockStsClient{
				FnGetCallerIdentity: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
					return &sts.GetCallerIdentityOutput{
						UserId: aws.String("user-123"),
					}, nil
				},
				FnAssumeRole: func(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error) {
					return nil, errors.New("failed to assume role")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentials, err := convertLocalCredsToSession(ctx, tt.mockStsClient, roleArn)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, credentials)
				require.Equal(t, "access-key-id", *credentials.AccessKeyId)
				require.Equal(t, "secret-access-key", *credentials.SecretAccessKey)
				require.Equal(t, "session-token", *credentials.SessionToken)
			}
		})
	}
}

func TestGetLatestAMIFromSSM(t *testing.T) {
	generateParamOutput := func(value string) *ssm.GetParameterOutput {
		return &ssm.GetParameterOutput{Parameter: &ssmTypes.Parameter{Value: &value}}
	}

	type returnedValues struct {
		output *ssm.GetParameterOutput
		err    error
	}

	tests := []struct {
		name           string
		parameterName  string
		returnedValues returnedValues
		wantErr        bool
		wantValue      string
	}{
		{
			name:          "successful parameter retrieval",
			parameterName: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			returnedValues: returnedValues{
				output: generateParamOutput("ami-12345678"),
				err:    nil,
			},
			wantErr:   false,
			wantValue: "ami-12345678",
		},
		{
			name:          "failed to get parameter",
			parameterName: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			returnedValues: returnedValues{
				output: nil,
				err:    errors.New("failed to get parameter"),
			},
			wantErr: true,
		},
		{
			name:          "bad output from SSM - nil",
			parameterName: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			returnedValues: returnedValues{
				output: nil,
				err:    nil,
			},
			wantErr: true,
		},
		{
			name:          "bad output from SSM - nil parameter",
			parameterName: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			returnedValues: returnedValues{
				output: &ssm.GetParameterOutput{Parameter: nil},
				err:    nil,
			},
			wantErr: true,
		},
		{
			name:          "bad output from SSM - nil parameter value",
			parameterName: "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			returnedValues: returnedValues{
				output: &ssm.GetParameterOutput{Parameter: &ssmTypes.Parameter{}},
				err:    nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSM := &mockSSMClient{
				fnGetParameter: func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					return tt.returnedValues.output, tt.returnedValues.err
				},
			}

			amiID, err := getLatestAMIFromSSM(context.Background(), mockSSM, tt.parameterName)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, amiID)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantValue, amiID)
			}
		})
	}
}

func TestGetAMIArchitecture(t *testing.T) {
	type returnedValues struct {
		output *ec2.DescribeImagesOutput
		err    error
	}

	tests := []struct {
		name             string
		amiID            string
		returnedValues   returnedValues
		wantArchitecture string
		wantErr          bool
	}{
		{
			name:  "successful architecture retrieval",
			amiID: "ami-12345678",
			returnedValues: returnedValues{
				output: &ec2.DescribeImagesOutput{
					Images: []ec2Types.Image{{
						ImageId:      aws.String("ami-12345678"),
						Architecture: ec2Types.ArchitectureValuesX8664,
					}},
				},
				err: nil,
			},
			wantArchitecture: string(ec2Types.ArchitectureValuesX8664),
			wantErr:          false,
		},
		{
			name:  "ec2 describe images error",
			amiID: "ami-12345678",
			returnedValues: returnedValues{
				output: nil,
				err:    errors.New("api error"),
			},
			wantErr: true,
		},
		{
			name:  "no images found",
			amiID: "ami-12345678",
			returnedValues: returnedValues{
				output: &ec2.DescribeImagesOutput{Images: []ec2Types.Image{}},
				err:    nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEC2 := &mockEC2Client{
				fnDescribeImages: func(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
					return tt.returnedValues.output, tt.returnedValues.err
				},
			}

			architecture, err := getAMIArchitecture(context.Background(), mockEC2, tt.amiID)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, architecture)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantArchitecture, architecture)
			}
		})
	}
}

func TestGetSupportedInstanceTypesSuccessful(t *testing.T) {
	instanceTypes := []ec2Types.InstanceTypeInfo{
		{
			InstanceType: ec2Types.InstanceTypeT2Micro,
			ProcessorInfo: &ec2Types.ProcessorInfo{
				SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
			},
		},
		{
			InstanceType: ec2Types.InstanceTypeT2Small,
			ProcessorInfo: &ec2Types.ProcessorInfo{
				SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
			},
		},
	}

	hasMorePages := true

	paginator := &mockInstanceTypesPaginator{
		instanceTypes: instanceTypes,
		fnNextPage: func(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
			hasMorePages = false
			return &ec2.DescribeInstanceTypesOutput{
				InstanceTypes: instanceTypes,
			}, nil
		},
		fnHasMorePages: func() bool {
			return hasMorePages
		},
	}

	got, err := getSupportedInstanceTypes(context.Background(), paginator, "x86_64")
	require.NoError(t, err)
	require.Equal(t, []string{"t2.micro", "t2.small"}, got)
}

func TestGetSupportedInstanceTypesPaginationError(t *testing.T) {
	hasMorePages := true
	paginator := &mockInstanceTypesPaginator{
		err: errors.New("pagination failed"),
		fnNextPage: func(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
			hasMorePages = false
			return nil, errors.New("pagination failed")
		},
		fnHasMorePages: func() bool {
			return hasMorePages
		},
	}

	got, err := getSupportedInstanceTypes(context.Background(), paginator, "x86_64")
	require.EqualError(t, err, "failed to load next pages for instance types: pagination failed")
	require.Nil(t, got)
}

func TestGetSupportedInstanceTypesNoMatching(t *testing.T) {
	instanceTypes := []ec2Types.InstanceTypeInfo{
		{
			InstanceType: ec2Types.InstanceTypeT2Micro,
			ProcessorInfo: &ec2Types.ProcessorInfo{
				SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
			},
		},
	}

	hasMorePages := true

	paginator := &mockInstanceTypesPaginator{
		instanceTypes: instanceTypes,
		fnNextPage: func(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
			hasMorePages = false
			return &ec2.DescribeInstanceTypesOutput{
				InstanceTypes: instanceTypes,
			}, nil
		},
		fnHasMorePages: func() bool {
			return hasMorePages
		},
	}

	got, err := getSupportedInstanceTypes(context.Background(), paginator, "arm64")
	require.NoError(t, err)
	require.Equal(t, []string(nil), got)
}

func TestValidateAMIType(t *testing.T) {
	tests := []struct {
		name          string
		amiType       string
		nodeType      string
		ssmValue      string
		ssmErr        error
		ec2Arch       string
		ec2Err        error
		wantErr       bool
		instanceTypes []string
	}{
		{
			name:     "valid ami and node type",
			wantErr:  false,
			amiType:  "AL2_x86_64",
			nodeType: "t2.micro",
			ssmValue: "ami-12345678",
			ssmErr:   nil,
			ec2Arch:  "x86_64",
			ec2Err:   nil,
			instanceTypes: []string{
				"t2.micro",
				"t2.small",
			},
		},
		{
			name:     "invalid ami type",
			wantErr:  true,
			amiType:  "INVALID_AMI",
			nodeType: "t2.micro",
		},
		{
			name:     "failed to get AMI ID from SSM",
			wantErr:  true,
			amiType:  "AL2_x86_64",
			nodeType: "t2.micro",
			ssmErr:   errors.New("failed to get parameter"),
		},
		{
			name:     "failed to get AMI architecture",
			wantErr:  true,
			amiType:  "AL2_x86_64",
			nodeType: "t2.micro",
			ssmValue: "ami-12345678",
			ec2Err:   errors.New("failed to describe images"),
		},
		{
			name:     "node type not supported",
			wantErr:  true,
			amiType:  "AL2_x86_64",
			nodeType: "t2.large",
			ssmValue: "ami-12345678",
			ec2Arch:  "x86_64",
			instanceTypes: []string{
				"t2.micro",
				"t2.small",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceTypes := []ec2Types.InstanceTypeInfo{
				{
					InstanceType: ec2Types.InstanceTypeT2Micro,
					ProcessorInfo: &ec2Types.ProcessorInfo{
						SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
					},
				},
				{
					InstanceType: ec2Types.InstanceTypeT2Small,
					ProcessorInfo: &ec2Types.ProcessorInfo{
						SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
					},
				},
			}

			mockSSM := &mockSSMClient{
				fnGetParameter: func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					if tt.ssmErr != nil {
						return nil, tt.ssmErr
					}
					return &ssm.GetParameterOutput{
						Parameter: &ssmTypes.Parameter{
							Value: &tt.ssmValue,
						},
					}, nil
				},
			}
			mockEC2 := &mockEC2Client{
				fnDescribeImages: func(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
					if tt.ec2Err != nil {
						return nil, tt.ec2Err
					}
					return &ec2.DescribeImagesOutput{
						Images: []ec2Types.Image{
							{
								Architecture: ec2Types.ArchitectureValues(tt.ec2Arch),
							},
						},
					}, nil
				},
			}

			hasMorePages := true

			mockPaginator := &mockInstanceTypesPaginator{
				instanceTypes: instanceTypes,
				fnNextPage: func(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
					hasMorePages = false
					return &ec2.DescribeInstanceTypesOutput{
						InstanceTypes: instanceTypes,
					}, nil
				},
				fnHasMorePages: func() bool {
					return hasMorePages
				},
			}

			err := validateAMIType(context.Background(), tt.amiType, tt.nodeType, mockSSM, mockEC2, mockPaginator)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Begin Mock definitions
type mockCredentialsProvider struct {
	fnRetrieve func(ctx context.Context) (aws.Credentials, error)
}

func (m *mockCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	if m.fnRetrieve == nil {
		return aws.Credentials{}, errors.New("not implemented")
	}

	creds, err := m.fnRetrieve(ctx)
	if err != nil {
		return aws.Credentials{}, err
	}
	return creds, nil
}

type mockSSMClient struct {
	fnGetParameter func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
}

func (m *mockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.fnGetParameter != nil {
		return m.fnGetParameter(ctx, input, opts...)
	}

	return nil, errors.New("not implemented")
}

type mockInstanceTypesPaginator struct {
	instanceTypes  []ec2Types.InstanceTypeInfo
	err            error
	fnHasMorePages func() bool
	fnNextPage     func(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error)
}

func (m *mockInstanceTypesPaginator) HasMorePages() bool {
	if m.fnHasMorePages != nil {
		return m.fnHasMorePages()
	}

	return false
}

func (m *mockInstanceTypesPaginator) NextPage(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	if m.fnNextPage != nil {
		return m.fnNextPage(ctx, opts...)
	}

	return nil, errors.New("not implemented")
}

type mockEC2Client struct {
	fnDescribeImages func(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

func (m *mockEC2Client) DescribeImages(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	if m.fnDescribeImages != nil {
		return m.fnDescribeImages(ctx, input, opts...)
	}

	return nil, errors.New("not implemented")
}
