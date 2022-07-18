package state

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
)

func EncryptFile(encryptedFilename string) error {

	key := []byte("asuperstrong32bitpasswordgohere!") //32-bit key for AES-256

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return err
	}

	cipherText := gcm.Seal(nonce, nonce, key, nil)

	err = ioutil.WriteFile(encryptedFilename, cipherText, 0777)
	if err != nil {
		return err
	}

	return nil
}
