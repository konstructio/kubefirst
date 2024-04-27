/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package main

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v.\n", err)
		fmt.Fprintf(os.Stderr, "\nIf a detailed error message was available, please make the necessary corrections before retrying.\nYou can re-run the last command to try the operation again.\n\n")

		os.Exit(1)
	}
}
