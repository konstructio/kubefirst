package ssh

import (
	"crypto/rand"
	"encoding/pem"

	"github.com/caarlos0/sshmarshal"
	goGitSsh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
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
