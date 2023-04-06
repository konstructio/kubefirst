/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helpers

import (
	"fmt"

	"github.com/spf13/afero"
)

// Use afero for file system to allow for easier testing
var fs afero.Fs = afero.NewOsFs()

// FileExists returns whether or not the given file exists in the OS
func FileExists(fs afero.Fs, filename string) bool {
	_, err := fs.Stat(filename)
	if err != nil {
		fmt.Println(err)
		return false
	}
	return true
}
