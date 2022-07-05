package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"github.com/spf13/viper"
	"os/exec"
	"syscall"
	"time"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	ssh2 "golang.org/x/crypto/ssh"
	"io/ioutil"
)

func createSoftServe(kubeconfigPath string) {
	if !viper.GetBool("create.softserve.create") {
		log.Println("Executing createSoftServe")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, createSoftServe skipped.")
			return
		}
		toolsDir := fmt.Sprintf("%s/.kubefirst/tools", home)

		err := os.Mkdir(toolsDir, 0777)
		if err != nil {
			log.Println("error creating directory %s", toolsDir, err)
		}

		// create soft-serve stateful set
		softServePath := fmt.Sprintf("%s/.kubefirst/gitops/components/soft-serve/manifests.yaml", home)
		softServeApplyOut, softServeApplyErr,errSoftServeApply := execShellReturnStrings(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "apply", "-f", softServePath, "--wait")
		log.Printf("Result:\n\t%s\n\t%s\n",softServeApplyOut,softServeApplyErr)	
		if errSoftServeApply != nil {
			log.Panicf("failed to call kubectlCreateSoftServeCmd.Run(): %v", err)
		}	

		viper.Set("create.softserve.create", true)
		viper.WriteConfig()
		log.Println("waiting for soft-serve installation to complete...")
		time.Sleep(60 * time.Second)
		//TODO: Update mechanism of waiting
	} else {
		log.Println("Skipping: createSoftServe")
	}

}

func configureSoftserveAndPush(){
	configureAndPushFlag := viper.GetBool("create.softserve.configure")
	if configureAndPushFlag != true {
		log.Println("Executing configureSoftserveAndPush")
		if dryrunMode {
			log.Printf("[#99] Dry-run mode, configureSoftserveAndPush skipped.")
			return
		}		
		kPortForward := exec.Command(kubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Println("failed to call kPortForward.Run(): ", err)
		}
		time.Sleep(10 * time.Second)

		configureSoftServe()
		pushGitopsToSoftServe()
		viper.Set("create.softserve.configure", true)
		viper.WriteConfig()
		time.Sleep(10 * time.Second)
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
		llog.Panicf("error pushing to remote", err)
	}

}
