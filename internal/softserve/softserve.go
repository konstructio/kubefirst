package softserve

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kubefirst/kubefirst/internal/gitClient"
	internalSSH "github.com/kubefirst/kubefirst/internal/ssh"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/gitlab"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
)

func CreateSoftServe(dryRun bool, kubeconfigPath string) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.softserve.create") {
		log.Println("creating soft-serve")
		if dryRun {
			log.Printf("[#99] Dry-run mode, createSoftServe skipped.")
			return
		}

		softServePath := fmt.Sprintf("%s/gitops/components/soft-serve/manifests.yaml", config.K1FolderPath)
		softServeApplyOut, softServeApplyErr, errSoftServeApply := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "apply", "-f", softServePath, "--wait")
		log.Printf("Result:\n\t%s\n\t%s\n", softServeApplyOut, softServeApplyErr)
		if errSoftServeApply != nil {
			log.Panicf("error: failed to apply soft-serve to the cluster %s", errSoftServeApply)
		}

		viper.Set("create.softserve.create", true)
		viper.WriteConfig()

	} else {
		log.Println("Skipping: createSoftServe")
	}

}

// ConfigureSoftServeAndPush calls Configure SoftServer and push gitops repository to GitLab
func ConfigureSoftServeAndPush(dryRun bool) error {

	if dryRun {
		log.Printf("[#99] Dry-run mode, configureSoftserveAndPush skipped.")
		return nil
	}

	config := configs.ReadConfig()

	configureAndPushFlag := viper.GetBool("create.softserve.configure")

	// soft serve is already configured, skipping
	if configureAndPushFlag {
		log.Println("Skipping: configureSoftserveAndPush")
		return nil
	}

	log.Println("Executing configureSoftserveAndPush")

	totalAttempts := 5
	for i := 0; i < totalAttempts; i++ {

		log.Printf("Configuring SoftServe, attempt (%d of %d)", i+1, totalAttempts)

		err := configureSoftServe()
		if err != nil {
			log.Printf("something went wrong, error is: %v, going to try again...", err)
			time.Sleep(10 * time.Second)
			continue
		}

		gitlab.PushGitRepo(dryRun, config, "soft", "gitops")

		viper.Set("create.softserve.configure", true)
		err = viper.WriteConfig()
		if err != nil {
			log.Printf("something went wrong, error is: %v, going to try again...", err)
			time.Sleep(10 * time.Second)
			continue
		}

		log.Println("waiting SoftServe to finish configuration, sleeping...")
		time.Sleep(30 * time.Second)

		log.Println("SoftServe successfully configured")
		return nil
	}

	return fmt.Errorf("we tried hard to setup SoftServe but there were something wrong, please check the logs")

}

// configureSoftServe clones local repositories, update config.yaml local file and commit push to Soft Serve
func configureSoftServe() error {
	config := configs.ReadConfig()

	url := pkg.SoftServerURI
	directory := fmt.Sprintf("%s/config", config.K1FolderPath)

	log.Println("gitClient clone", url, directory)

	auth, err := internalSSH.PublicKey()
	if err != nil {
		return err
	}

	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	repo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("error cloning config repository from soft serve, error: %v", err)
	}

	file, err := os.ReadFile(fmt.Sprintf("%s/config.yaml", directory))
	if err != nil {
		return fmt.Errorf("error reading config.yaml file %s", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = os.WriteFile(fmt.Sprintf("%s/config.yaml", directory), []byte(newFile), 0)
	if err != nil {
		return err
	}

	log.Printf("re-wrote config.yaml at %s/config folder", config.K1FolderPath)

	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	log.Println("Committing new changes...")
	_ = gitClient.GitAddWithFilter(viper.GetString("cloud"), "gitops", w)
	_, err = w.Commit("updating soft-serve server config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: config.InstallerEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return fmt.Errorf("error pushing to remote, error: %v", err)
	}

	return nil

}
