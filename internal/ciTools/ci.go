package ciTools

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/flagset"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/internal/repo"
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

	err := SedBucketName("<BUCKET_NAME>", bucketName)
	if err != nil {
		log.Panicf("Error sed bucket name on CI repository: %s", err)
		return err
	}

	ciLocation := fmt.Sprintf("%s/ci/components/argo-gitlab/ci.yaml", config.K1FolderPath)

	DetokenizeCI("<CI_CLUSTER_NAME>", viper.GetString("ci.cluster.name"), ciLocation)
	DetokenizeCI("<CI_S3_SUFFIX>", viper.GetString("ci.s3.suffix"), ciLocation)
	DetokenizeCI("<CI_HOSTED_ZONE_NAME>", viper.GetString("ci.hosted.zone.name"), ciLocation)

	gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "ci")

	//if !viper.GetBool("gitlab.ci-pushed") {
	//	log.Println("Pushing ci repo to origin gitlab")
	//	gitlab.PushGitRepo(globalFlags.DryRun, config, "gitlab", "ci")
	//	viper.Set("gitlab.ci-pushed", true)
	//	viper.WriteConfig()
	//	log.Println("clone and detokenization of ci-template repository complete")
	//}

	return nil
}

func SedBucketName(old, new string) error {
	cfg := configs.ReadConfig()
	providerFile := fmt.Sprintf("%s/ci/terraform/base/provider.tf", cfg.K1FolderPath)

	fileData, err := ioutil.ReadFile(providerFile)
	if err != nil {
		return err
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = ioutil.WriteFile(providerFile, fileData, 0o600)
	if err != nil {
		return err
	}

	return nil
}

//func CopyCIYamlToGitlab(globalFlags flagset.GlobalFlags) error {
//	cfg := configs.ReadConfig()
//
//	if globalFlags.DryRun {
//		log.Printf("[#99] Dry-run mode, DeployOnGitlab skipped.")
//		return nil
//	}
//
//	oldLocation := fmt.Sprintf("%s/ci/components/argo-gitlab/ci.yaml", cfg.K1FolderPath)
//
//	DetokenizeCI("<CI_CLUSTER_NAME>", viper.GetString("ci.cluster.name"), oldLocation)
//	DetokenizeCI("<CI_S3_SUFFIX>", viper.GetString("ci.s3.suffix"), oldLocation)
//	DetokenizeCI("<CI_HOSTED_ZONE_NAME>", viper.GetString("ci.hosted.zone.name"), oldLocation)
//
//	newLocation := fmt.Sprintf("%s/gitops/components/argo-gitlab/ci.yaml", cfg.K1FolderPath)
//	newRepository := fmt.Sprintf("%s/gitops", cfg.K1FolderPath)
//	err := os.Rename(oldLocation, newLocation)
//	if err != nil {
//		return err
//	}
//
//	repo, err := git.PlainOpen(newRepository)
//	if err != nil {
//		log.Printf("error opening the directory %s:  %s", newRepository, err)
//		return err
//	}
//
//	w, err := repo.Worktree()
//	if err != nil {
//		log.Printf("error to make worktree:  %s", err)
//		return err
//	}
//
//	auth := &gitHttp.BasicAuth{
//		Username: "root",
//		Password: viper.GetString("gitlab.token"),
//	}
//
//	err = w.Pull(&git.PullOptions{
//		RemoteName: "gitlab",
//		Auth:       auth,
//	})
//	if err != nil {
//		log.Print(err)
//	}
//
//	_, err = w.Add("components/argo-gitlab/ci.yaml")
//	if err != nil {
//		log.Printf("error to add:  %s", err)
//		return err
//	}
//	_, err = w.Commit(fmt.Sprint("committing detokenized ci yaml file"), &git.CommitOptions{
//		Author: &object.Signature{
//			Name:  "kubefirst-bot",
//			Email: "kubefirst-bot@kubefirst.com",
//			When:  time.Now(),
//		},
//	})
//	if err != nil {
//		log.Printf("error to commit:  %s", err)
//		return err
//	}
//
//	err = repo.Push(&git.PushOptions{
//		RemoteName: "gitlab",
//		Auth:       auth,
//		Force:      true,
//	})
//	if err != nil {
//		log.Println("error pushing to remote", err)
//		return err
//	}
//	return nil
//}

func DetokenizeCI(old, new, ciLocation string) {
	ciFile := ciLocation

	fileData, err := ioutil.ReadFile(ciFile)
	if err != nil {
		//return err
		log.Println(err)
	}

	fileString := string(fileData)
	fileString = strings.ReplaceAll(fileString, old, new)
	fileData = []byte(fileString)

	err = ioutil.WriteFile(ciFile, fileData, 0o600)
	if err != nil {
		//return err
		log.Println(err)
	}
	// return nil
}
