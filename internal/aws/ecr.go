package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

func (conf *AWSConfiguration) GetECRAuthToken() (string, error) {
	fmt.Println("getting ecr auth token")
	ecrClient := ecr.NewFromConfig(conf.Config)

	token, err := ecrClient.GetAuthorizationToken(context.Background(), &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}

	return *token.AuthorizationData[0].AuthorizationToken, nil
}
