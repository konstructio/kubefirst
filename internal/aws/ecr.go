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

func (conf *AWSConfiguration) GetECRRepositoryURL(clusterID string, repoName string) (string, error) {
	fmt.Printf("checking ecr repositories for existing repo %s", repoName)
	ecrClient := ecr.NewFromConfig(conf.Config)

	repos, err := ecrClient.DescribeRepositories(context.TODO(), &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{repoName},
	})

	if err != nil {
		return "", err
	}

	//*look up the /metaphor repository

	//* if it exists, append the clusterID to the

	for _, repo := range repos.Repositories {
		fmt.Println("repo name is: ", *repo.CreatedAt)
	}
	return "", nil
}
