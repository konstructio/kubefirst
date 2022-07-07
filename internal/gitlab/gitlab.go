package gitlab

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/google/uuid"
	"github.com/kubefirst/nebulous/configs"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/crypto/ssh"
)

// GenerateKey generate public and private keys to be consumed by GitLab.
func GenerateKey() (string, string, error) {
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

func GitlabGeneratePersonalAccessToken(gitlabPodName string) {
	config := configs.ReadConfig()

	kPortForward := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "port-forward", "svc/gitlab-webservice-default", "8888:8080")
	kPortForward.Stdout = os.Stdout
	kPortForward.Stderr = os.Stderr
	err := kPortForward.Start()
	defer kPortForward.Process.Signal(syscall.SIGTERM)
	if err != nil {
		log.Panicf("error: failed to port-forward to gitlab %s", err)
	}

	log.Println("generating gitlab personal access token on pod: ", gitlabPodName)

	id := uuid.New()
	gitlabToken := id.String()[:20]

	k := exec.Command(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "-n", "gitlab", "exec", gitlabPodName, "--", "gitlab-rails", "runner", fmt.Sprintf("token = User.find_by_username('root').personal_access_tokens.create(scopes: [:write_registry, :write_repository, :api], name: 'Automation token'); token.set_token('%s'); token.save!", gitlabToken))
	k.Stdout = os.Stdout
	k.Stderr = os.Stderr
	err = k.Run()
	if err != nil {
		log.Panicf("error running exec against %s to generate gitlab personal access token for root user", gitlabPodName)
	}

	viper.Set("gitlab.token", gitlabToken)
	viper.WriteConfig()

	log.Println("gitlab personal access token generated", gitlabToken)
}
