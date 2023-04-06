/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import "github.com/aws/aws-sdk-go-v2/aws"

// AWSConfiguration stores session data to organize all AWS functions into a single struct
type AWSConfiguration struct {
	Config aws.Config
}
