package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
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
