/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func (conf *AWSConfiguration) GetCallerIdentity() (*sts.GetCallerIdentityOutput, error) {

	stsClient := sts.NewFromConfig(conf.Config)
	iamCaller, err := stsClient.GetCallerIdentity(
		context.Background(),
		&sts.GetCallerIdentityInput{},
	)
	if err != nil {
		fmt.Printf("error: could not get caller identity %s", err)
	}
	return iamCaller, nil
}
