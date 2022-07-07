package softserve

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/gitClient"
	"github.com/kubefirst/nebulous/pkg"
	"github.com/spf13/viper"
	ssh2 "golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func CreateSoftServe(kubeconfigPath string) {
	config := configs.ReadConfig()
	if !viper.GetBool("create.softserve.create") {
		log.Println("Executing CreateSoftServe")
		if config.DryRun {
			log.Printf("[#99] Dry-run mode, CreateSoftServe skipped.")
			return
		}
		toolsDir := fmt.Sprintf("%s/.kubefirst/tools", config.HomePath)

		err := os.Mkdir(toolsDir, 0777)
		if err != nil {
			log.Println("error creating directory %s", toolsDir, err)
		}

		// create soft-serve stateful set
		softServePath := fmt.Sprintf("%s/.kubefirst/gitops/components/soft-serve/manifests.yaml", config.HomePath)
		softServeApplyOut, softServeApplyErr, errSoftServeApply := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", kubeconfigPath, "-n", "soft-serve", "apply", "-f", softServePath, "--wait")
		log.Printf("Result:\n\t%s\n\t%s\n", softServeApplyOut, softServeApplyErr)
		if errSoftServeApply != nil {
			log.Panicf("failed to call kubectlCreateSoftServeCmd.Run(): %v", err)
		}

		viper.Set("create.softserve.create", true)
		viper.WriteConfig()
		log.Println("waiting for soft-serve installation to complete...")
		time.Sleep(60 * time.Second)
		//TODO: Update mechanism of waiting
	} else {
		log.Println("Skipping: CreateSoftServe")
	}

}

func ConfigureSoftServeAndPush() {
	config := configs.ReadConfig()
	configureAndPushFlag := viper.GetBool("create.softserve.configure")
	if configureAndPushFlag != true {
		log.Println("Executing ConfigureSoftServeAndPush")
		if config.DryRun {
			log.Printf("[#99] Dry-run mode, ConfigureSoftServeAndPush skipped.")
			return
		}
		kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "soft-serve", "port-forward", "svc/soft-serve", "8022:22")
		kPortForward.Stdout = os.Stdout
		kPortForward.Stderr = os.Stderr
		err := kPortForward.Start()
		defer kPortForward.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Panicf("error: failed to port-forward to soft-serve %s", err)
		}
		time.Sleep(20 * time.Second)

		configureSoftServe()
		gitClient.PushGitopsToSoftServe()
		viper.Set("create.softserve.configure", true)
		viper.WriteConfig()
		time.Sleep(30 * time.Second)
	} else {
		log.Println("Skipping: ConfigureSoftServeAndPush")
	}
}

func configureSoftServe() {
	config := configs.ReadConfig()

	url := "ssh://127.0.0.1:8022/config"
	directory := fmt.Sprintf("%s/.kubefirst/config", config.HomePath)

	log.Println("gitClient clone", url, directory)

	auth, _ := pkg.PublicKey()

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
		log.Panic(err)
	}

	println("re-wrote config.yaml", config.HomePath, "/.kubefirst/config")

	w, _ := repo.Worktree()

	log.Println("Committing new changes...")
	w.Add(".")
	w.Commit("updating soft-serve server config", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kubefirst-bot",
			Email: config.InstallerEmail,
			When:  time.Now(),
		},
	})

	err = repo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		log.Panicf("error pushing to remote", err)
	}

}
