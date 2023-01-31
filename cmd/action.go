package cmd

import (
	"context"
	"fmt"

	cssv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:   "action",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		var c client.Client

		css := &cssv1beta1.ClusterSecretStore{}
		c.Get(context.Background(), client.ObjectKey{
			Namespace: "external-secrets-operator",
			Name:      "vault-secrets-backend",
		}, css)

		fmt.Println(css)

		//!

		// var clusterSecretStore ClusterSecretStore

		// clusterName := "k1-vault-spike"
		// k1DirPath := viper.GetString("kubefirst.k1-directory-path")
		// helmClientPath := viper.GetString("kubefirst.helm-client-path")
		// k1GitopsDir := viper.GetString("kubefirst.k1-gitops-dir")
		// kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")

		// kubectlClientPath := viper.GetString("kubefirst.kubectl-client-path")
		// // registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", k1DirPath, clusterName)
		// registryYamlPath := "https://raw.githubusercontent.com/jarededwards/vault-spike/main/registry/k1-vault-spike/registry.yaml"

		// // //* helm repo add and update
		// // helmRepo := helm.HelmRepo{
		// // 	RepoName:     "argo",
		// // 	RepoURL:      "https://argoproj.github.io/argo-helm",
		// // 	ChartName:    "argo-cd",
		// // 	Namespace:    "argocd",
		// // 	ChartVersion: "5.19.12",
		// // }

		// // fmt.Println("adding helm repo and update")
		// // helm.AddRepoAndUpdateRepo(false, helmClientPath, helmRepo, kubeconfigPath)

		// // //* helm install argocd
		// // fmt.Println("helm install argocd")
		// // err := helm.Install(false, helmClientPath, helmRepo, kubeconfigPath)
		// // if err != nil {
		// // 	return err
		// // }

		// //* create external-secrets-operator ns
		// //* k8s secret for cluster store connectivity
		// clientset, err := k8s.GetClientSet(false, kubeconfigPath)
		// if err != nil {
		// 	log.Info().Msg("error getting kubernetes clientset")
		// }

		// newNamespaces := []string{"external-secrets-operator"}
		// for i, s := range newNamespaces {
		// 	namespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: s}}
		// 	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, metav1.CreateOptions{})
		// 	if err != nil {
		// 		log.Error().Err(err).Msg("")
		// 		return errors.New("error creating namespace")
		// 	}
		// 	log.Info().Msgf("%d, %s", i, s)
		// 	log.Info().Msgf("namespace created: %s", s)
		// }

		// vaultData := map[string][]byte{
		// 	"vault-token": []byte("k1_local_vault_token"),
		// }
		// vaultSecret := &v1.Secret{
		// 	ObjectMeta: metav1.ObjectMeta{Name: "vault-token", Namespace: "external-secrets-operator"},
		// 	Data:       vaultData,
		// }
		// _, err = clientset.CoreV1().Secrets("external-secrets-operator").Create(context.TODO(), vaultSecret, metav1.CreateOptions{})
		// if err != nil {
		// 	log.Error().Err(err).Msg("")
		// 	return errors.New("error creating kubernetes secret: external-secrets-operator/vault-token")
		// }

		// clientset.BatchV1()

		// // //* kubectl apply -f registry yaml
		// _, _, err = pkg.ExecShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "argocd", "apply", "-f", registryYamlPath, "--wait")
		// if err != nil {
		// 	log.Info().Msgf("failed to execute kubectl apply -f %s: error %s", registryYamlPath, err.Error())
		// 	return err
		// }

		// // todo k8s.GetPodStatus
		// time.Sleep(45 * time.Second)

		// //* vault port-forward
		// vaultStopChannel := make(chan struct{}, 1)
		// defer func() {
		// 	close(vaultStopChannel)
		// }()
		// k8s.OpenPortForwardPodWrapper(
		// 	kubeconfigPath,
		// 	"vault-0",
		// 	"vault",
		// 	8200,
		// 	8200,
		// 	vaultStopChannel,
		// )

		// //* configure vault with terraform
		// //* run vault terraform
		// tfEnvs := map[string]string{}
		// tfEnvs = terraform.GetVaultSpikeTerraformEnvs(tfEnvs)
		// tfEntrypoint := k1GitopsDir + "/terraform/vault-spike"
		// err = terraform.InitApplyAutoApprove(false, tfEntrypoint, tfEnvs)
		// if err != nil {
		// 	return err
		// }

		//! DELETE create vault configured secret
		// // todo remove this code
		// log.Info().Msg("creating vault configured secret")
		// k8s.CreateVaultConfiguredSecret(false, kubeconfigPath, kubectlClientPath)
		// viper.Set("terraform.vault.apply.complete", true)
		// viper.WriteConfig()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
