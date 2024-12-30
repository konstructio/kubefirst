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
	"github.com/stretchr/testify/assert"
)

type mockCredentialsProvider struct {
	creds aws.Credentials
	err   error
}

func (m *mockCredentialsProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return m.creds, m.err
}

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
			err:         nil,
			expectedErr: nil,
		},
		{
			name:        "failed to retrieve credentials",
			creds:       aws.Credentials{},
			err:         errors.New("failed to retrieve credentials"),
			expectedErr: errors.New("failed to retrieve AWS credentials: failed to retrieve credentials"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProvider := &mockCredentialsProvider{
				creds: tt.creds,
				err:   tt.err,
			}

			creds, err := getSessionCredentials(context.Background(), mockProvider)
			if tt.expectedErr != nil {
				assert.Nil(t, creds)
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NotNil(t, creds)
				assert.NoError(t, err)
				assert.Equal(t, tt.creds, *creds)
			}
		})
	}
}

func TestGetLatestAMIFromSSM(t *testing.T) {
	tests := []struct {
		name           string
		parameterName  string
		parameterValue string
		err            error
		expectedErr    error
	}{
		{
			name:           "successful parameter retrieval",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "ami-12345678",
			err:            nil,
			expectedErr:    nil,
		},
		{
			name:           "failed to get parameter",
			parameterName:  "/aws/service/eks/optimized-ami/1.29/amazon-linux-2/recommended/image_id",
			parameterValue: "",
			err:            errors.New("parameter not found"),
			expectedErr:    errors.New("failed to initialise ssm client: parameter not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSM := &mockSSMClient{
				parameterValue: tt.parameterValue,
				err:            tt.err,
			}

			amiID, err := GetLatestAMIFromSSM(context.Background(), mockSSM, tt.parameterName)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
				assert.Empty(t, amiID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.parameterValue, amiID)
			}
		})
	}
}

func TestGetAMIArchitecture(t *testing.T) {
	tests := []struct {
		name         string
		amiID        string
		architecture string
		images       []ec2Types.Image
		err          error
		expectedErr  string
	}{
		{
			name:         "successful architecture retrieval",
			amiID:        "ami-12345678",
			architecture: "x86_64",
			images: []ec2Types.Image{
				{
					Architecture: ec2Types.ArchitectureValuesX8664,
				},
			},
			err:         nil,
			expectedErr: "",
		},
		{
			name:         "ec2 describe images error",
			amiID:        "ami-12345678",
			architecture: "",
			images:       nil,
			err:          errors.New("api error"),
			expectedErr:  "failed to describe images: api error",
		},
		{
			name:         "no images found",
			amiID:        "ami-12345678",
			architecture: "",
			images:       []ec2Types.Image{},
			err:          nil,
			expectedErr:  "no images found for AMI ID: ami-12345678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEC2 := &mockEC2Client{
				images: tt.images,
				err:    tt.err,
			}

			architecture, err := GetAMIArchitecture(context.Background(), mockEC2, tt.amiID)
			fmt.Printf("arch is %s\n", string(architecture))
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Empty(t, architecture)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.architecture, architecture)
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

			got, err := GetSupportedInstanceTypes(context.Background(), paginator, tt.architecture)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
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

type mockSSMClient struct {
	ssm.Client
	parameterValue string
	err            error
}

func (m *mockSSMClient) GetParameter(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &ssm.GetParameterOutput{
		Parameter: &ssmTypes.Parameter{
			Value: &m.parameterValue,
		},
	}, nil
}

type mockEC2Client struct {
	ec2.Client
	images       []ec2Types.Image
	architecture string
	err          error
}

func (m *mockEC2Client) DescribeImages(ctx context.Context, input *ec2.DescribeImagesInput, opts ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.architecture == "x86_64" {
		return &ec2.DescribeImagesOutput{
			Images: []ec2Types.Image{
				{
					Architecture: ec2Types.ArchitectureValues("x86_64"),
				},
			},
		}, nil
	} else if m.architecture == "" {
		return &ec2.DescribeImagesOutput{
			Images: m.images,
		}, nil
	} else {
		return nil, errors.New("unexpected architecture")
	}
}

// func TestValidateAMIType(t *testing.T) {
// 	tests := []struct {
// 		name          string
// 		amiType       string
// 		nodeType      string
// 		ssmValue      string
// 		ssmErr        error
// 		ec2Arch       string
// 		ec2Err        error
// 		expectedErr   error
// 		instanceTypes []string
// 	}{
// 		{
// 			name:        "valid ami and node type",
// 			amiType:     "AL2_x86_64",
// 			nodeType:    "t2.micro",
// 			ssmValue:    "ami-12345678",
// 			ssmErr:      nil,
// 			ec2Arch:     "x86_64",
// 			ec2Err:      nil,
// 			expectedErr: nil,
// 			instanceTypes: []string{
// 				"t2.micro",
// 				"t2.small",
// 			},
// 		},
// 		{
// 			name:        "invalid ami type",
// 			amiType:     "INVALID_AMI",
// 			nodeType:    "t2.micro",
// 			expectedErr: errors.New("not a valid ami type"),
// 		},
// 		{
// 			name:        "failed to get AMI ID from SSM",
// 			amiType:     "AL2_x86_64",
// 			nodeType:    "t2.micro",
// 			ssmErr:      errors.New("failed to get parameter"),
// 			expectedErr: errors.New("failed to get AMI ID from SSM: failed to get parameter"),
// 		},
// 		{
// 			name:        "failed to get AMI architecture",
// 			amiType:     "AL2_x86_64",
// 			nodeType:    "t2.micro",
// 			ssmValue:    "ami-12345678",
// 			ec2Err:      errors.New("failed to describe images"),
// 			expectedErr: errors.New("failed to get AMI architecture: failed to describe images"),
// 		},
// 		{
// 			name:        "node type not supported",
// 			amiType:     "AL2_x86_64",
// 			nodeType:    "t2.large",
// 			ssmValue:    "ami-12345678",
// 			ec2Arch:     "x86_64",
// 			expectedErr: errors.New("node type t2.large not supported for AL2_x86_64"),
// 			instanceTypes: []string{
// 				"t2.micro",
// 				"t2.small",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// t.Run(tt.name, func(t *testing.T) {
// mockSSM := &mockSSMClient{
// 	parameterValue: tt.ssmValue,
// 	err:            tt.ssmErr,
// }
// mockEC2 := &mockEC2Client{
// 	architecture: tt.ec2Arch,
// 	err:          tt.ec2Err,
// }

// // Mock the GetLatestAMIFromSSM and GetAMIArchitecture functions
// GetLatestAMIFromSSM = func(ctx context.Context, ssmClient *ssm.Client, parameterName string) (string, error) {
// 	return mockSSM.parameterValue, mockSSM.err
// }
// GetAMIArchitecture = func(ctx context.Context, ec2Client *ec2.Client, amiID string) (string, error) {
// 	return mockEC2.architecture, mockEC2.err
// }
// GetSupportedInstanceTypes = func(cfg aws.Config, architecture string) ([]string, error) {
// 	return tt.instanceTypes, nil
// }

// err := GetLatestAMIFromSSM(context.Background(), aws.Config{}, tt.amiType, tt.nodeType)
// if tt.expectedErr != nil {
// 	assert.EqualError(t, err, tt.expectedErr.Error())
// } else {
// 	assert.NoError(t, err)
// }
// })
// 	}
// }
