package cmd

import (
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/spf13/cobra"
)

func NewDevCommand() *cobra.Command {
	devCommand := &cobra.Command{
		Use:   "dev",
		Short: "",
		RunE:  runDev,
	}
	return devCommand
}

func runDev(cmd *cobra.Command, args []string) error {

	err := k8s.IngressCreate("vault", "vault", 8200)
	if err != nil {
		return err
	}
	//err := k8s.IngressDelete("vault", "vault")
	//if err != nil {
	//	return err
	//}
	//err := k8s.IngressAddRule("default", "k3d-ingress-rules", "vault", 8200)
	//if err != nil {
	//	return err
	//}

	return nil
}
