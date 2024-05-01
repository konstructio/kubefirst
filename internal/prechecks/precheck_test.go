package prechecks

import (
	"testing"
)

// test envvar exists
func TestEnvVarExists(t *testing.T) {
	if EnvVarExists("TEST") {
		t.Fatal("TEST env var should not exist")
	}

	t.Setenv("ENV", "123")

	if !EnvVarExists("ENV") {
		t.Fatal("ENV env var should exist")
	}
}

// test url is available
func TestURLIsAvailable(t *testing.T) {
	if URLIsAvailable("google.com:443") != nil {
		t.Fatal("Google should be reachable")
	}
}

// test file exists
func TestFileExists(t *testing.T) {
	if FileExists("test") == nil {
		t.Fatal("File should not exist")
	}
}

// test check available disk size
func TestCheckAvailableDiskSize(t *testing.T) {
	if CheckAvailableDiskSize() != nil {
		t.Fatal("Disk size should be available")
	}
}

// test command exists
func TestCommandExists(t *testing.T) {
	if err := CommandExists("ls"); err != nil {
		t.Fatal("ls command should exist")
	}
}
