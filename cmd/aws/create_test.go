package aws

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmTypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/stretchr/testify/require"
)

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name        string
		creds       aws.Credentials
		err         error
		expectedErr error
	}{
		{
			name: "valid credentials",
			creds: aws.Credentials{
				AccessKeyID:     "test-access-key-id",
				SecretAccessKey: "test-secret-access-key",
				SessionToken:    "test-session-token",
			},
			err: nil,
		},
		{
			name:  "failed to retrieve credentials",
			creds: aws.Credentials{},
			err:   errors.New("failed to retrieve credentials"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockCredentialsProvider{
				fnRetrieve: func(ctx context.Context) (aws.Credentials, error) {
					return tt.creds, tt.err
				},
			}

			creds, err := getSessionCredentials(context.Background(), mockProvider)

			if tt.err != nil {
				require.ErrorContains(t, err, tt.err.Error())
			} else {
				require.NotNil(t, creds)
				require.NoError(t, err)
				require.Equal(t, tt.creds, *creds)
			}
		})
	}
}

func TestGetLatestAMIFromSSM(t *testing.T) {
	tests := []struct {
		name           string
		parameterName  string
		parameterValue string
		wantErr        bool
		err            error
		fnGetParameter func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	}{
		{
			name:           "successful parameter retrieval",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "ami-12345678",
			wantErr:        false,
			err:            nil,
		},
		{
			name:           "failed to get parameter",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "",
			wantErr:        true,
			err:            errors.New("failed to get parameter"),
		},
		{
			name:           "bad output from SSM - nil",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "",
			wantErr:        true,
			fnGetParameter: func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return nil, nil
			},
		},
		{
			name:           "bad output from SSM - nil parameter",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "",
			wantErr:        true,
			fnGetParameter: func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return &ssm.GetParameterOutput{
					Parameter: nil,
				}, nil
			},
		},
		{
			name:           "bad output from SSM - nil parameter value",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "",
			wantErr:        true,
			fnGetParameter: func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
				return &ssm.GetParameterOutput{
					Parameter: &ssmTypes.Parameter{},
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fnGetParameter == nil {
				tt.fnGetParameter = func(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
					if tt.err != nil {
						return nil, tt.err
					}
					return &ssm.GetParameterOutput{
						Parameter: &ssmTypes.Parameter{
							Value: &tt.parameterValue,
						},
					}, nil
				}
			}

			mockSSM := &mockSSMClient{
				fnGetParameter: tt.fnGetParameter,
			}

			amiID, err := getLatestAMIFromSSM(context.Background(), mockSSM, tt.parameterName)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, amiID)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.parameterValue, amiID)
			}
		})
	}
}

func TestGetAMIArchitecture(t *testing.T) {
	tests := []struct {
		name             string
		wantErr          bool
		amiID            string
		architecture     string
		images           []ec2Types.Image
		err              error
		fnDescribeImages func(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
	}{
		{
			name:         "successful architecture retrieval",
			wantErr:      false,
			amiID:        "ami-12345678",
			architecture: "x86_64",
			images: []ec2Types.Image{
				{
					Architecture: ec2Types.ArchitectureValuesX8664,
				},
			},
			err: nil,
		},
		{
			name:         "ec2 describe images error",
			wantErr:      true,
			amiID:        "ami-12345678",
			architecture: "",
			images:       nil,
			err:          errors.New("api error"),
		},
		{
			name:         "no images found",
			wantErr:      true,
			amiID:        "ami-12345678",
			architecture: "",
			images:       []ec2Types.Image{},
			err:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fnDescribeImages == nil {
				tt.fnDescribeImages = func(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
					if tt.err != nil {
						return nil, tt.err
					}
					return &ec2.DescribeImagesOutput{
						Images: tt.images,
					}, nil
				}
			}

			mockEC2 := &mockEC2Client{
				fnDescribeImages: tt.fnDescribeImages,
			}

			architecture, err := getAMIArchitecture(context.Background(), mockEC2, tt.amiID)
			fmt.Printf("arch is %q\n", string(architecture))
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, architecture)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.architecture, architecture)
			}
		})
	}
}

func TestGetSupportedInstanceTypes(t *testing.T) {
	tests := []struct {
		name          string
		architecture  string
		instanceTypes []ec2Types.InstanceTypeInfo
		paginateErr   error
		expected      []string
		expectedErr   error
	}{
		{
			name:         "successful instance types retrieval",
			architecture: "x86_64",
			instanceTypes: []ec2Types.InstanceTypeInfo{
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
			},
			paginateErr: nil,
			expected:    []string{"t2.micro", "t2.small"},
			expectedErr: nil,
		},
		{
			name:         "pagination error",
			architecture: "x86_64",
			paginateErr:  errors.New("pagination failed"),
			expected:     nil,
			expectedErr:  fmt.Errorf("failed to load next pages for instance types: pagination failed"),
		},
		{
			name:         "no matching instance types",
			architecture: "arm64",
			instanceTypes: []ec2Types.InstanceTypeInfo{
				{
					InstanceType: ec2Types.InstanceTypeT2Micro,
					ProcessorInfo: &ec2Types.ProcessorInfo{
						SupportedArchitectures: []ec2Types.ArchitectureType{ec2Types.ArchitectureTypeX8664},
					},
				},
			},
			paginateErr: nil,
			expected:    []string(nil),
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paginator := &mockInstanceTypesPaginator{
				instanceTypes: tt.instanceTypes,
				err:           tt.paginateErr,
			}

			got, err := getSupportedInstanceTypes(context.Background(), paginator, tt.architecture)
			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, got)
			}
		})
	}
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

			err := validateAMIType(context.Background(), tt.amiType, tt.nodeType, mockSSM, mockEC2, &mockInstanceTypesPaginator{
				instanceTypes: []ec2Types.InstanceTypeInfo{
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
				},
				err:    nil,
				called: false,
			})
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
	instanceTypes []ec2Types.InstanceTypeInfo
	err           error
	called        bool
}

func (m *mockInstanceTypesPaginator) HasMorePages() bool {
	return !m.called
}

func (m *mockInstanceTypesPaginator) NextPage(ctx context.Context, opts ...func(*ec2.Options)) (*ec2.DescribeInstanceTypesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.called = true
	return &ec2.DescribeInstanceTypesOutput{
		InstanceTypes: m.instanceTypes,
	}, nil
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
