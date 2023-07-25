package generate

import (
	"fmt"

	"github.com/spf13/cobra"
)

func generate(cmd *cobra.Command, args []string) error {
	fmt.Println("hello generate. lets do this")
	return nil
}
