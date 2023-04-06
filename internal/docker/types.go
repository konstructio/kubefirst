/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package docker

import "github.com/docker/docker/client"

type DockerClientWrapper struct {
	Client *client.Client
}
