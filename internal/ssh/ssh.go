package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/sshmarshal"
	goGitSsh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

func CreateSshKeyPair() (string, string, error) {

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

	return privateKey, publicKey, nil
}

func PublicKeyV2() (*goGitSsh.PublicKeys, error) {
	var publicKey *goGitSsh.PublicKeys
	publicKey, err := goGitSsh.NewPublicKeys("kube1st", []byte(viper.GetString("kubefirst.bot.private-key")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}

func PublicKey() (*goGitSsh.PublicKeys, error) {
	var publicKey *goGitSsh.PublicKeys
	publicKey, err := goGitSsh.NewPublicKeys("gitClient", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}

// todo hack - need something more substantial and accommodating and not in ssh..
func WriteGithubArgoCdInitValuesFile(githubGitopsSshURL, k1Dir, sshPrivateKey string) error {

	var argocdInitValuesYaml = []byte(fmt.Sprintf(`
configs:
  repositories:
    gitops:
      url: %s/gitops.git
      type: gitClient
      name: gitops
  credentialTemplates:
    ssh-creds:
      url: %s
      sshPrivateKey: |
        %s
`, githubGitopsSshURL, githubGitopsSshURL, strings.ReplaceAll(sshPrivateKey, "\n", "\n        ")))

	err := os.WriteFile(fmt.Sprintf("%s/argocd-init-values.yaml", k1Dir), argocdInitValuesYaml, 0644)
	if err != nil {
		log.Info().Msgf("error: could not write %s/argocd-init-values.yaml %s", k1Dir, err)
		return err
	}
	return nil
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
