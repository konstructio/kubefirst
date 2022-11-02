package pkg

import "syscall"

func GetDiskFree() (uint64, error) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs("/", &fs)
	if err != nil {
		return 0, err
	}
	return fs.Bfree * uint64(fs.Bsize), nil
}
