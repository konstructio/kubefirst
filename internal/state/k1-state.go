package state

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"io/ioutil"
	"os"
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

func SendFileToS3(bucketName string, filename string) error {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("sa-east-1"),
	})

	file, err := os.Open(filename)
	if err != nil {
		exitErrorf("Unable to open file %q, %v", filename, err)
		return err
	}

	defer file.Close()

	uploader := s3manager.NewUploader(sess)

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String("test-1.txt"),
		Body:   file,
	})
	if err != nil {
		// Print the error and exit.
		exitErrorf("Unable to upload %q to %q, %v", filename, bucketName, err)
		return err
	}

	fmt.Printf("Successfully uploaded %q to %q\n", filename, bucketName)

	return nil

}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
