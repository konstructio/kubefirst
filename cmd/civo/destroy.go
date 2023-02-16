package civo

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/civo/civogo"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/internal/terraform"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func destroyCivo(cmd *cobra.Command, args []string) error {

	log.Info().Msg("destroying kubefirst platform in civo")

	adminEmail := viper.GetString("flags.admin-email")
	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	dryRun := false
	githubOwner := viper.GetString("github.owner")
	k1Dir := viper.GetString("k1-paths.k1-dir")
	k1GitopsDir := viper.GetString("k1-paths.gitops-dir")
	kubefirstConfigPath := viper.GetString("k1-paths.kubefirst-config")
	registryYamlPath := fmt.Sprintf("%s/gitops/registry/%s/registry.yaml", k1Dir, clusterName)

	// todo improve these checks, make them standard for
	// both create and destroy
	githubToken := os.Getenv("GITHUB_TOKEN")
	civoToken := os.Getenv("CIVO_TOKEN")
	if len(githubToken) == 0 {
		return errors.New("ephemeral tokens not supported for cloud installations, please set a GITHUB_TOKEN environment variable to continue\n https://docs.kubefirst.io/kubefirst/github/install.html#step-3-kubefirst-init")
	}
	if len(civoToken) == 0 {
		return errors.New("\n\nYour CIVO_TOKEN environment variable isn't set,\nvisit this link https://dashboard.civo.com/security and set the environment variable")
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-github") {
		log.Info().Msg("destroying github resources with terraform")

		tfEntrypoint := k1GitopsDir + "/terraform/github"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
		err := terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("kubefirst-checks.terraform-apply-github", false)
		viper.WriteConfig()
		log.Info().Msg("github resources terraform destroyed")
	}

	if viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		log.Info().Msg("destroying civo resources with terraform")

		clusterName := viper.GetString("flags.cluster-name")
		kubeconfigPath := viper.GetString("k1-paths.kubeconfig")
		region := viper.GetString("flags.cloud-region")

		client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
		if err != nil {
			log.Info().Msg(err.Error())
			return err
		}

		cluster, err := client.FindKubernetesCluster(clusterName)
		if err != nil {
			return err
		}
		log.Info().Msg("cluster name: " + cluster.ID)

		clusterVolumes, err := client.ListVolumesForCluster(cluster.ID)
		if err != nil {
			return err
		}

		log.Info().Msg("opening argocd port forward")
		//* ArgoCD port-forward
		argoCDStopChannel := make(chan struct{}, 1)
		defer func() {
			close(argoCDStopChannel)
		}()
		k8s.OpenPortForwardPodWrapper(
			kubeconfigPath,
			"argocd-server",
			"argocd",
			8080,
			8080,
			argoCDStopChannel,
		)

		log.Info().Msg("getting new auth token for argocd")
		argocdAuthToken, err := argocd.GetArgoCDToken(viper.GetString("components.argocd.username"), viper.GetString("components.argocd.password"))
		if err != nil {
			return err
		}

		log.Info().Msgf("port-forward to argocd is available at %s", viper.GetString("components.argocd.port-forward-url"))

		customTransport := http.DefaultTransport.(*http.Transport).Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		argocdHttpClient := http.Client{Transport: customTransport}
		log.Info().Msg("deleting the registry application")
		httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, registryYamlPath, argocdAuthToken, "true")
		if err != nil {
			return err
		}
		log.Info().Msgf("http status code %d", httpCode)

		for _, vol := range clusterVolumes {
			log.Info().Msg("removing volume with name: " + vol.Name)
			_, err := client.DeleteVolume(vol.ID)
			if err != nil {
				return err
			}
			log.Info().Msg("volume " + vol.ID + " deleted")
		}

		log.Info().Msg("destroying civo cloud resources")
		tfEntrypoint := k1GitopsDir + "/terraform/civo"
		tfEnvs := map[string]string{}
		tfEnvs = civo.GetCivoTerraformEnvs(tfEnvs)
		tfEnvs = civo.GetGithubTerraformEnvs(tfEnvs)
		err = terraform.InitDestroyAutoApprove(dryRun, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			return err
		}
		viper.Set("kubefirst-checks.terraform-apply-civo", false)
		viper.WriteConfig()
		log.Info().Msg("civo resources terraform destroyed")
	}

	//* remove local content and kubefirst config file for re-execution
	if !viper.GetBool("kubefirst-checks.terraform-apply-github") && !viper.GetBool("kubefirst-checks.terraform-apply-civo") {
		log.Info().Msg("removing previous platform content")

		err := pkg.ResetK1Dir(k1Dir, kubefirstConfigPath)
		if err != nil {
			return err
		}
		log.Info().Msg("previous platform content removed")

		log.Info().Msg("resetting `$HOME/.kubefirst` config")
		viper.Set("argocd", "")
		viper.Set("github", "")
		viper.Set("components", "")
		viper.Set("kbot", "")
		viper.Set("kubefirst-checks", "")
		viper.Set("kubefirst", "")
		viper.WriteConfig()
	}

	fmt.Println("\nsuccessfully removed previous platform content from `$HOME/.k1`")
	fmt.Println("to recreate your kubefirst platform run")
	fmt.Println("\n\nkubefirst civo create \\")
	fmt.Printf("    --admin-email %s \\\n", adminEmail)
	fmt.Printf("    --domain-name %s \\\n", domainName)
	fmt.Printf("    --github-owner %s \\\n", githubOwner)
	fmt.Printf("    --cluster-name %s"+"\n"+"\n"+"\n", clusterName)
	return nil
}
