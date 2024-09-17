package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

func Checkstatus() error {

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return fmt.Errorf("Docker is not installed or not accessible: %w", err)
	}

	_, err = cli.Ping(context.Background())
	if err != nil {
		return fmt.Errorf("Docker is not accessible: %v", err)
	}

	return nil
}
