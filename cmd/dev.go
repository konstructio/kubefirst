package cmd

import (
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/spf13/cobra"
	"log"
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

	config := configs.ReadConfig()

	// create local certs using MKCert tool
	log.Println("Installing CA from MkCert")
	ssl.InstallCALocal(config)
	log.Println("Creating local certs using MkCert")
	ssl.CreateCertsLocal(config)

	// todo: add remaining apps
	appListForCertificate := []string{"argocd", "argo, vault"}
	log.Println("creating local certificates")
	if err := ssl.CreateCertificatesForLocalWrapper(config, appListForCertificate); err != nil {
		log.Println(err)
	}
	log.Println("creating local certificates done")

	log.Println("storing certificates into application secrets namespace")
	if err := k8s.CreateSecretsFromCertificatesForLocalWrapper(config, appListForCertificate); err != nil {
		log.Println(err)
	}
	log.Println("storing certificates into application secrets namespace done")

	//argoCDConfig := argocd.Config{}
	//// Repo config
	//gitOpsRepo := fmt.Sprintf("git@%s:%s/gitops.git", viper.GetString("github.host"), viper.GetString("github.owner"))
	//
	//argoCDConfig.Configs.Repositories.RepoGitops.URL = gitOpsRepo
	//argoCDConfig.Configs.Repositories.RepoGitops.Type = "git"
	//argoCDConfig.Configs.Repositories.RepoGitops.Name = "github-gitops"
	//
	//// Credentials
	//argoCDConfig.Configs.CredentialTemplates.SSHCreds.URL = gitOpsRepo
	//argoCDConfig.Configs.CredentialTemplates.SSHCreds.SSHPrivateKey = viper.GetString("botprivatekey")
	//
	//// Ingress
	//argoCDConfig.Server.ExtraArgs = []string{"--insecure"}
	//argoCDConfig.Server.Ingress.Enabled = "true"
	//argoCDConfig.Server.Ingress.Annotations.IngressKubernetesIoRewriteTarget = "/"
	//argoCDConfig.Server.Ingress.Annotations.IngressKubernetesIoBackendProtocol = "HTTPS"
	//argoCDConfig.Server.Ingress.Hosts = []string{"argocd.localhost"}
	//
	//argoCDConfig.Server.Ingress.TLS = append(argoCDConfig.Server.Ingress.TLS, argocd.TLSConfig{Hosts: []string{"argocd.localhost"}, SecretName: "argocd-secret"})
	//
	//config := configs.ReadConfig()
	//err := argocd.CreateInitialArgoCDRepository(config, argoCDConfig)
	//if err != nil {
	//	return err
	//}

	//// todo: add Thiago's path
	//privKey, err := pkg.GetFileContent("./cert.pem")
	//if err != nil {
	//	return err
	//}
	//// todo: add Thiago's path
	//pubKey, err := pkg.GetFileContent("./key.pem")
	//if err != nil {
	//	return err
	//}
	//
	//data := map[string][]byte{
	//	"privKey": privKey,
	//	"pubKey":  pubKey,
	//}
	//
	//err = k8s.CreateSecret("vault", "vault-tls", data)
	//if err != nil {
	//	return err
	//}

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
