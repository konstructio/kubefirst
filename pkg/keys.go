package pkg

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/kubefirst/kubefirst/configs"
	"log"
	"os"
	"strings"

	"github.com/caarlos0/sshmarshal"
	goGitSsh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

func CreateSshKeyPair() {

	publicKey := viper.GetString("botpublickey")

	// generate GitLab keys
	if publicKey == "" && viper.GetString("gitprovider") == "gitlab" {

		log.Println("generating new key pair for GitLab")
		publicKey, privateKey, err := generateGitLabKeys()
		if err != nil {
			log.Println(err)
		}

		viper.Set("botpublickey", publicKey)
		viper.Set("botprivatekey", privateKey)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}
	}

	// generate GitHub keys
	if publicKey == "" && viper.GetString("gitprovider") == "github" {

		log.Println("generating new key pair for GitHub")
		publicKey, privateKey, err := generateGitHubKeys()
		if err != nil {
			log.Println(err)
		}

		viper.Set("botpublickey", publicKey)
		viper.Set("botprivatekey", privateKey)
		err = viper.WriteConfig()
		if err != nil {
			log.Panicf("error: could not write to viper config")
		}

	}
	publicKey = viper.GetString("botpublickey")

	// todo: break it into smaller function
	if viper.GetString("gitprovider") != CloudK3d {

		config := configs.ReadConfig()
		privateKey := viper.GetString("botprivatekey")

		var argocdInitValuesYaml = []byte(fmt.Sprintf(`
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

		err := os.WriteFile(fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), argocdInitValuesYaml, 0644)
		if err != nil {
			log.Panicf("error: could not write argocd-init-values.yaml %s", err)
		}
	}
}

func PublicKey() (*goGitSsh.PublicKeys, error) {
	var publicKey *goGitSsh.PublicKeys
	publicKey, err := goGitSsh.NewPublicKeys("gitClient", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}

// generateGitLabKeys generate public and private keys to be consumed by GitLab. Private Key is encrypted using RSA key with
// SHA-1
func generateGitLabKeys() (string, string, error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return "", "", err
	}

	pub, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return "", "", err
	}
	publicKey := string(ssh.MarshalAuthorizedKey(pub))
	// encode RSA key
	privateKey := string(pem.EncodeToMemory(
		&pem.Block{
			Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	))

	return publicKey, privateKey, nil
}

// generateGitHubKeys generate Public and Private ED25519 keys for GitHub.
func generateGitHubKeys() (string, string, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}

	ecdsaPublicKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", err
	}

	pemPrivateKey, err := sshmarshal.MarshalPrivateKey(privKey, "kubefirst key")
	if err != nil {
		return "", "", err
	}

	privateKey := string(pem.EncodeToMemory(pemPrivateKey))
	publicKey := string(ssh.MarshalAuthorizedKey(ecdsaPublicKey))

	return publicKey, privateKey, nil
}

// todo: function not in use, can we remove it?
func ModConfigYaml() {

	file, err := os.ReadFile("./config.yaml")
	if err != nil {
		log.Println("error reading file", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = os.WriteFile("./config.yaml", []byte(newFile), 0)
	if err != nil {
		panic(err)
	}
}
