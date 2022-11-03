package pkg

import "syscall"

// GetAvailableDiskSize returns the available disk size in the user machine. In that way Kubefirst can validate
// if the available disk size is enough to start a installation.
func GetAvailableDiskSize() (uint64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs("/", &fs)
	if err != nil {
		return 0, err
	}
	return fs.Bfree * uint64(fs.Bsize), nil
}
