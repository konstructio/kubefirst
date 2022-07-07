package pkg

import (
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/nebulous/configs"
	"github.com/kubefirst/nebulous/internal/gitlab"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"strings"
)

func CreateSshKeyPair() {
	config := configs.ReadConfig()
	publicKey := viper.GetString("botpublickey")
	if publicKey == "" {
		log.Println("generating new key pair")
		publicKey, privateKey, _ := gitlab.GenerateKey()
		viper.Set("botPublicKey", publicKey)
		viper.Set("botPrivateKey", privateKey)
		err := viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}
	publicKey = viper.GetString("botpublickey")
	privateKey := viper.GetString("botprivatekey")

	var argocdInitValuesYaml = []byte(fmt.Sprintf(`
server:
  additionalApplications:
  - name: registry
    namespace: argocd
    additionalLabels: {}
    additionalAnnotations: {}
    finalizers:
    - resources-finalizer.argocd.argoproj.io
    project: default
    source:
      repoURL: ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops
      targetRevision: HEAD
      path: registry
    destination:
      server: https://kubernetes.default.svc
      namespace: argocd
    syncPolicy:
      automated:
        prune: true
        selfHeal: true
      syncOptions:
      - CreateNamespace=true
configs:
  repositories:
    soft-serve-gitops:
      url: ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops
      insecure: 'true'
      type: gitClient
      name: soft-serve-gitops
  credentialTemplates:
    ssh-creds:
      url: ssh://soft-serve.soft-serve.svc.cluster.local:22
      sshPrivateKey: |
        %s
`, strings.ReplaceAll(privateKey, "\n", "\n        ")))

	err := ioutil.WriteFile(fmt.Sprintf("%s/.kubefirst/argocd-init-values.yaml", config.HomePath), argocdInitValuesYaml, 0644)
	if err != nil {
		log.Panicf("error: could not write argocd-init-values.yaml %s", err)
	}
}

func PublicKey() (*ssh.PublicKeys, error) {
	var publicKey *ssh.PublicKeys
	publicKey, err := ssh.NewPublicKeys("gitClient", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}
