package docker

import "github.com/docker/docker/client"

type DockerClientWrapper struct {
	Client *client.Client
}
