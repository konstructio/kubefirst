/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
)

func (conf *AWSConfiguration) GetKmsKeyID(keyAlias string) (string, error) {

	var kmsKeyId string
	kmsClient := kms.NewFromConfig(conf.Config)

	kmsKeys, err := kmsClient.ListAliases(context.Background(), &kms.ListAliasesInput{})
	if err != nil {
		log.Info().Msgf("error: could not list kms key aliases %s", err)
	}

	for _, k := range kmsKeys.Aliases {
		if *k.AliasName == keyAlias {
			log.Info().Msgf("kms key with alias %s found", *k.AliasName)
			log.Info().Msgf("kms key id for vault dynamodb is: %s", *k.TargetKeyId)
			kmsKeyId = *k.TargetKeyId
		}
	}

	return kmsKeyId, nil
}
