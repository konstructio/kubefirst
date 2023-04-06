/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func (conf *AWSConfiguration) GetIamRole(roleName string) (*iam.GetRoleOutput, error) {

	// fmt.Println("looking up iam role: ", roleName) // todo add helpful logs about if found or not
	iamClient := iam.NewFromConfig(conf.Config)

	role, err := iamClient.GetRole(context.Background(), &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
	if err != nil {
		return &iam.GetRoleOutput{}, err
	}

	return role, nil
}
