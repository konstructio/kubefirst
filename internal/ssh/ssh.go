package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/caarlos0/sshmarshal"
	goGitSsh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/argocd"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"os"
)

func CreateSshKeyPair() {

	publicKey := viper.GetString("botpublickey")

	// generate GitLab keys
	if publicKey == "" && viper.GetString("gitprovider") == "gitlab" {

		log.Info().Msg("generating new key pair for GitLab")
		publicKey, privateKey, err := generateGitLabKeys()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		viper.Set("botpublickey", publicKey)
		viper.Set("botprivatekey", privateKey)
		err = viper.WriteConfig()
		if err != nil {
			log.Panic().Msg("error: could not write to viper config")
		}
	}

	// generate GitHub keys
	if publicKey == "" && viper.GetString("gitprovider") == "github" {

		log.Info().Msg("generating new key pair for GitHub")
		publicKey, privateKey, err := generateGitHubKeys()
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		viper.Set("botpublickey", publicKey)
		viper.Set("botprivatekey", privateKey)
		err = viper.WriteConfig()
		if err != nil {
			log.Panic().Msg("error: could not write to viper config")
		}

	}
	publicKey = viper.GetString("botpublickey")

	// todo: break it into smaller function
	if viper.GetString("gitprovider") != pkg.CloudK3d {

		config := configs.ReadConfig()
		privateKey := viper.GetString("botprivatekey")

		argoCDConfig := argocd.Config{}
		argoCDConfig.Configs.Repositories.SoftServeGitops.URL = "ssh://soft-serve.soft-serve.svc.cluster.local:22/gitops"
		argoCDConfig.Configs.Repositories.Insecure = "true"
		argoCDConfig.Configs.Repositories.Type = "gitClient"
		argoCDConfig.Configs.Repositories.Name = "soft-serve-gitops"
		argoCDConfig.Configs.CredentialTemplates.SSHCreds.URL = "ssh://soft-serve.soft-serve.svc.cluster.local:22"
		argoCDConfig.Configs.CredentialTemplates.SSHCreds.SSHPrivateKey = privateKey

		argoData, err := yaml.Marshal(&argoCDConfig)
		if err != nil {
			log.Panic().Err(err).Msg("")
		}

		err = os.WriteFile(fmt.Sprintf("%s/argocd-init-values.yaml", config.K1FolderPath), argoData, 0644)
		if err != nil {
			log.Panic().Msgf("error: could not write argocd-init-values.yaml %s", err)
		}
	}
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

func PublicKey() (*goGitSsh.PublicKeys, error) {
	var publicKey *goGitSsh.PublicKeys
	publicKey, err := goGitSsh.NewPublicKeys("gitClient", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		log.Panic().Err(err).Msg("error: could not write to viper config")
	}
	return publicKey, err
}
