package k8s

import (
	"fmt"
	"net"
)

// CheckForExistingPortForwards determines whether or not port forwards are already running
// If so, a warning is issued
func CheckForExistingPortForwards(ports ...int) error {
	for _, port := range ports {
		listen, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%v", port))
		if err != nil {
			return fmt.Errorf("port %v is in use", port)
		}
		_ = listen.Close()
	}

	return nil
}
