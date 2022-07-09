package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
)

func createSoftServe() {
	if !viper.GetBool("create.softserve.create") {
		log.Println("creating soft-serve")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, createSoftServe skipped.")
			return
		}

		softServePath := fmt.Sprintf("%s/.kubefirst/gitops/components/soft-serve/manifests.yaml", home)
		softServeApplyOut, softServeApplyErr, errSoftServeApply := execShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "apply", "-f", softServePath, "--wait")
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

func configureSoftserveAndPush() {
	configureAndPushFlag := viper.GetBool("create.softserve.configure")

	if !configureAndPushFlag {
		log.Println("Executing configureSoftserveAndPush")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, configureSoftserveAndPush skipped.")
			return
		}

		configureSoftServe()
		pushGitRepo("soft", "gitops")

		viper.Set("create.softserve.configure", true)
		viper.WriteConfig()
		time.Sleep(30 * time.Second)
	} else {
		log.Println("Skipping: configureSoftserveAndPush")
	}
}

func configureSoftServe() {
	url := "ssh://127.0.0.1:8022/config"
	directory := fmt.Sprintf("%s/.kubefirst/config", home)

	log.Println("git clone", url, directory)

	auth, _ := publicKey()

	auth.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	repo, err := git.PlainClone(directory, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		log.Panicf("error cloning config repository from soft serve")
	}

	file, err := ioutil.ReadFile(fmt.Sprintf("%s/config.yaml", directory))
	if err != nil {
		log.Panicf("error reading config.yaml file %s", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = ioutil.WriteFile(fmt.Sprintf("%s/config.yaml", directory), []byte(newFile), 0)
	if err != nil {
		panic(err)
	}

	println("re-wrote config.yaml", home, "/.kubefirst/config")

	w, _ := repo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("updating soft-serve server config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: installerEmail,
			When:  time.Now(),
		},
	})

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		log.Panicf("error pushing to remote %s", err)
	}

}
