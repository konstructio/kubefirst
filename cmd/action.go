package cmd

import (
	"context"
	"fmt"
	"strings"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cm "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
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

		domainName := viper.GetString("domain-name")
		kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")
		k1Dir := viper.GetString("kubefirst.k1-dir")

		clientset, err := k8s.GetClientSet(false, kubeconfigPath)
		if err != nil {
			fmt.Println("error building rest config")
			return err
		}

		// secret resources
		secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("error listing secrets in all namespaces")
			return err
		}

		fmt.Println(domainName, k1Dir)

		for _, secret := range secrets.Items {
			if strings.Contains(secret.Name, "-tls") {
				fmt.Println("backing up secret (ns/resource): " + secret.Namespace + "/" + secret.Name)
				secret.SetManagedFields(nil)
				secret.SetOwnerReferences(nil)
				secret.SetAnnotations(nil)
				secret.SetCreationTimestamp(metav1.Time{})

				fmt.Println(secret.APIVersion)

				fileName := fmt.Sprintf("%s/ssl/%s/secrets/%s-%s.yaml", k1Dir, domainName, secret.Namespace, secret.Name)
				fmt.Printf("writing file: %s\n\n", fileName)
				yamlContent, err := yaml.Marshal(secret)
				if err != nil {
					return fmt.Errorf("unable to marshal yaml: %s", err)
				}
				pkg.CreateFile(fileName, yamlContent)

			} else {
				fmt.Println("skipping secret: ", secret.Name)
			}
		}

		// cert manager resources
		k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		cmClientSet, err := cm.NewForConfig(k8sConfig)
		if err != nil {
			fmt.Println("error building cert manager cmClientSet")
			return err
		}

		certs, err := cmClientSet.CertmanagerV1().Certificates("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("error getting list of certificates")
		}

		for _, cert := range certs.Items {
			if strings.Contains(cert.Name, "-tls") {
				fmt.Println("backing up certificate (ns/resource): " + cert.Namespace + "/" + cert.Name)
				cert.SetManagedFields(nil)
				cert.SetOwnerReferences(nil)
				cert.SetAnnotations(nil)
				cert.SetResourceVersion("")
				cert.Status = cmv1.CertificateStatus{}
				cert.SetCreationTimestamp(metav1.Time{})
				// cert.SetUID() // todo couldn't get nil value

				fileName := fmt.Sprintf("%s/ssl/%s/certificates/%s-%s.yaml", k1Dir, domainName, cert.Namespace, cert.Name)
				fmt.Printf("writing file: %s\n", fileName)
				yamlContent, err := yaml.Marshal(cert)
				if err != nil {
					return fmt.Errorf("unable to marshal yaml: %s", err)
				}
				pkg.CreateFile(fileName, yamlContent)
			} else {
				fmt.Println("skipping certficate (ns/resource): " + cert.Namespace + "/" + cert.Name)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(actionCmd)
}
