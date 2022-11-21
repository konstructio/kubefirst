package cmd

import (
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
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

	// todo: add Thiago's path
	privKey, err := pkg.GetFileContent("./cert.pem")
	if err != nil {
		return err
	}
	// todo: add Thiago's path
	pubKey, err := pkg.GetFileContent("./key.pem")
	if err != nil {
		return err
	}

	data := map[string][]byte{
		"privKey": privKey,
		"pubKey":  pubKey,
	}

	err = k8s.CreateSecret("vault", "vault-tls", data)
	if err != nil {
		return err
	}

	//err := k8s.IngressCreate("vault", "vault", 8200)
	//if err != nil {
	//	return err
	//}
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
