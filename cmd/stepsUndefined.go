package cmd


import (
	"log"
	"strings"
	"github.com/spf13/viper"
	"io/ioutil"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func modConfigYaml() {

	file, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		log.Println("error reading file", err)
	}

	newFile := strings.Replace(string(file), "allow-keyless: false", "allow-keyless: true", -1)

	err = ioutil.WriteFile("./config.yaml", []byte(newFile), 0)
	if err != nil {
		panic(err)
	}
}



func publicKey() (*ssh.PublicKeys, error) {
	var publicKey *ssh.PublicKeys
	publicKey, err := ssh.NewPublicKeys("git", []byte(viper.GetString("botprivatekey")), "")
	if err != nil {
		return nil, err
	}
	return publicKey, err
}