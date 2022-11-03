package ciTools

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
)

// DeployOnGitlab - Deploy CI applications on gitlab install
func DeployOnGitlab(globalFlags flagset.GlobalFlags, bucketName string) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, DeployOnGitlab skipped.")
		return nil
	}
	log.Printf("cloning and detokenizing the ci-template repository")
	repo.PrepareKubefirstTemplateRepo(globalFlags.DryRun, config, viper.GetString("gitops.owner"), "ci", viper.GetString("ci.branch"), viper.GetString("template.tag"))
	log.Println("clone and detokenization of ci-template repository complete")

	secretProviderFile := fmt.Sprintf("%s/ci/terraform/secret/provider.tf", config.K1FolderPath)
	baseProviderFile := fmt.Sprintf("%s/ci/terraform/base/provider.tf", config.K1FolderPath)

	err := SedBucketName("<BUCKET_NAME>", bucketName, secretProviderFile)
	if err != nil {
		log.Panicf("Error sed bucket name on CI repository: %s", err)
		return err
	}

	err = SedBucketName("<BUCKET_NAME>", bucketName, baseProviderFile)
	if err != nil {
		log.Panicf("Error sed bucket name on CI repository: %s", err)
		return err
	}

	ciLocation := ""
	workflowLocation := fmt.Sprintf("%s/ci/.gitlab-ci.yml", config.K1FolderPath)

	if viper.GetString("ci.flavor") == "github" {
		ciLocation = fmt.Sprintf("%s/ci/components/argo-github/ci.yaml", config.K1FolderPath)
	} else {
		ciLocation = fmt.Sprintf("%s/ci/components/argo-gitlab/ci.yaml", config.K1FolderPath)
	}

	err = DetokenizeCI("<CI_GITOPS_BRANCH>", viper.GetString("ci.gitops.branch"), ciLocation)
	if err != nil {
		log.Println(err)
	}

	err = DetokenizeCI("<CI_METAPHOR_BRANCH>", viper.GetString("ci.metaphor.branch"), ciLocation)
	if err != nil {
		log.Println(err)
	}

	err = DetokenizeCI("<CI_CLUSTER_NAME>", viper.GetString("ci.cluster.name"), ciLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<CI_S3_SUFFIX>", viper.GetString("ci.s3.suffix"), ciLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<CI_HOSTED_ZONE_NAME>", viper.GetString("ci.hosted.zone.name"), ciLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<FLAVOR>", viper.GetString("ci.flavor"), workflowLocation)
	if err != nil {
		log.Println(err)
	}
	err = DetokenizeCI("<CI_KUBEFIRST_BRANCH>", viper.GetString("ci.kubefirst.branch"), workflowLocation)
	if err != nil {
		log.Println(err)
	}

	if viper.GetString("ci.flavor") == "github" {
		err = DetokenizeCI("<CI_GITHUB_USER>", viper.GetString("ci.github.user"), ciLocation)
		if err != nil {
			log.Println(err)
		}
		err = DetokenizeCI("<CI_GITHUB_OWNER>", viper.GetString("ci.github.owner"), ciLocation)
		if err != nil {
			log.Println(err)
		}
	}

	if !viper.GetBool("gitlab.ci-pushed") {
		log.Println("Pushing ci repo to origin gitlab")
		gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "ci")
		viper.Set("gitlab.ci-pushed", true)
		viper.WriteConfig()
		log.Println("clone and detokenization of ci-template repository complete")
	}

	return nil
}

func SedBucketName(old string, new string, providerFile string) error {
	fileData, err := os.ReadFile(providerFile)
	if err != nil {
		return err
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = os.WriteFile(providerFile, fileData, 0o600)
	if err != nil {
		return err
	}

	return nil
}

func DetokenizeCI(old, new, ciLocation string) error {
	ciFile := ciLocation

	fileData, err := os.ReadFile(ciFile)
	if err != nil {
		return err
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = os.WriteFile(ciFile, fileData, 0o600)
	if err != nil {
		return err
	}
	return nil
}

func DestroyGitRepository(globalFlags flagset.GlobalFlags) error {
	domain := viper.GetString("aws.hostedzonename")
	url := fmt.Sprintf("https://gitlab.%s/api/v4/projects/kubefirst%%2Fci", domain)
	_, _, err := pkg.ExecShellReturnStrings("curl", "-H", "-vL", "-X", "DELETE", url, "-H", "Content-Type: application/json", "-H", fmt.Sprintf("Private-Token: %s", viper.GetString("gitlab.token")))
	if err != nil {
		log.Panicf("error: delete CI repository: %s", err)
		return err
	}
	return nil
}

func ApplyTemplates(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, ApplyTemplates skipped.")
		return nil
	}

	_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argo", "apply", "-f", fmt.Sprintf("%s/ci/components/templates/cwft-k1-ci.yaml", config.K1FolderPath))
	if err != nil {
		log.Printf("failed to execute kubectl apply of cwft-k1-ci: %s", err)
		return err
	}

	time.Sleep(45 * time.Second)
	viper.Set("ci.cwft-k1-ci.applied", true)
	viper.WriteConfig()

	return nil
}

func DeleteTemplates(globalFlags flagset.GlobalFlags) error {
	config := configs.ReadConfig()
	if globalFlags.DryRun {
		log.Printf("[#99] Dry-run mode, ApplyTemplates skipped.")
		return nil
	}

	_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "argo", "delete", "-f", fmt.Sprintf("%s/ci/components/templates/cwft-k1-ci.yaml", config.K1FolderPath))
	if err != nil {
		log.Printf("failed to execute kubectl delete of cwft-k1-ci: %s", err)
		return err
	}

	time.Sleep(45 * time.Second)
	viper.Set("ci.cwft-k1-ci.deleted", true)
	viper.WriteConfig()

	return nil
}
