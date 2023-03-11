package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

func NewDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal().Msgf("error instantiating docker client: %s", err)
	}

	return cli
}

func (docker DockerClientWrapper) ListContainers() {
	containers, err := docker.Client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}
}

// CheckDockerReady
func (docker DockerClientWrapper) CheckDockerReady() (bool, error) {
	_, err := docker.Client.Info(context.Background())
	if err != nil {
		log.Error().Msgf("error determining docker readiness: %s", err)
		return false, err
	}

	return true, nil
}
