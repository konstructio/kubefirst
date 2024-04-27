package prechecks

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kubefirst/runtime/pkg"
	"github.com/kubefirst/runtime/pkg/docker"
	"github.com/kubefirst/runtime/pkg/ssh"
	"github.com/rs/zerolog/log"
)

// RequireEnvVarExists returns true if the given env var name exists.
func EnvVarExists(n string) bool {
	_, ok := os.LookupEnv(n)

	return ok
}

// URLIsAvailable returns true if the given URL is reachable.
func URLIsAvailable(url string) error {
	timeout := 3 * time.Second

	if _, err := net.DialTimeout("tcp", url, timeout); err != nil {
		return err
	}

	return nil
}

// FileExists returns an error if the given file does not exist.
func FileExists(f string) error {
	if _, err := os.Stat(f); err != nil {
		return fmt.Errorf("%s does not exist - but is required", f)
	}

	return nil
}

// CheckAvailableDiskSize returns an error if not enough disk space is available.
func CheckAvailableDiskSize() error {
	free, err := pkg.GetAvailableDiskSize()
	if err != nil {
		return err
	}

	// convert available disk size to GB format
	availableDiskSize := float64(free) / humanize.GByte
	if availableDiskSize < pkg.MinimumAvailableDiskSize {
		return fmt.Errorf("there is not enough space to proceed with the installation, a minimum of %d GB is required to proceed", pkg.MinimumAvailableDiskSize)
	}

	return nil
}

// CommandExists return true if the give command exists within the users $PATH variable.
func CommandExists(cmd string) error {
	if _, err := exec.LookPath(cmd); err != nil {
		return fmt.Errorf("%s not installed - is but required", cmd)
	}

	return nil
}

// CheckKnownHosts checks if the host is in the known_hosts file.
func CheckKnownHosts(url string) error {
	key, err := ssh.GetHostKey(url)
	if err != nil {
		return fmt.Errorf("known_hosts file does not exist - please run `ssh-keyscan %s >> ~/.ssh/known_hosts` to remedy", url)
	}

	log.Info().Msgf("%s %s\n", url, key.Type())

	return nil
}

// CheckDockerIsRunning makes sure Docker is running.
func CheckDockerIsRunning() error {
	dcli := docker.DockerClientWrapper{
		Client: docker.NewDockerClient(),
	}
	if _, err := dcli.CheckDockerReady(); err != nil {
		return fmt.Errorf("docker must be running to use this command. Error checking Docker status: %s", err)
	}

	return nil
}
