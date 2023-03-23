package civo

import (
	"os"

	"github.com/civo/civogo"
	"github.com/rs/zerolog/log"
)

func CreateStorageBucket(accessKeyId, bucketName, region string) (civogo.ObjectStore, error) {
	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return civogo.ObjectStore{}, err
	}

	bucket, err := client.NewObjectStore(&civogo.CreateObjectStoreRequest{
		Name:        bucketName,
		Region:      region,
		AccessKeyID: accessKeyId,
		MaxSizeGB:   500,
	})
	if err != nil {
		return civogo.ObjectStore{}, err
	}

	return *bucket, nil
}

// todo refactor or remove this internal library and use the native client. functionality. see next todo client.
func GetAccessCredentials(credentialName, region string) (civogo.ObjectStoreCredential, error) {

	creds, err := checkKubefirstCredentials(credentialName, region)
	if err != nil {
		log.Info().Msg(err.Error())
	}

	if creds == (civogo.ObjectStoreCredential{}) {
		log.Info().Msgf("credential name: %s not found, creating", credentialName)
		creds, err = createAccessCredentials(credentialName, region)
		if err != nil {
			return civogo.ObjectStoreCredential{}, err
		}

		creds, err = getAccessCredentials(creds.ID, region)
		if err != nil {
			return civogo.ObjectStoreCredential{}, err
		}

		log.Info().Msgf("created object storage credential %s", credentialName)
		return creds, nil
	}

	return creds, nil
}

func DeleteAccessCredentials(credentialName, region string) error {

	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return err
	}

	creds, err := checkKubefirstCredentials(credentialName, region)
	if err != nil {
		log.Info().Msg(err.Error())
	}

	_, err = client.DeleteObjectStoreCredential(creds.ID)
	if err != nil {
		return err
	}

	return nil
}

func checkKubefirstCredentials(credentialName, region string) (civogo.ObjectStoreCredential, error) {

	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return civogo.ObjectStoreCredential{}, err
	}

	// todo client.FindObjectStoreCredential()
	log.Info().Msgf("looking for credential: %s", credentialName)
	remoteCredentials, err := client.ListObjectStoreCredentials()
	if err != nil {
		log.Info().Msg(err.Error())
		return civogo.ObjectStoreCredential{}, err
	}

	var creds civogo.ObjectStoreCredential

	for i, cred := range remoteCredentials.Items {
		if cred.Name == credentialName {
			log.Info().Msgf("found credential: %s", credentialName)
			return remoteCredentials.Items[i], nil
		}
	}

	return creds, err
}

// todo client.NewObjectStoreCredential()
func createAccessCredentials(credentialName, region string) (civogo.ObjectStoreCredential, error) {

	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return civogo.ObjectStoreCredential{}, err
	}
	creds, err := client.NewObjectStoreCredential(&civogo.CreateObjectStoreCredentialRequest{
		Name:   credentialName,
		Region: region,
	})
	if err != nil {
		log.Info().Msg(err.Error())
	}
	return *creds, nil
}

func getAccessCredentials(id, region string) (civogo.ObjectStoreCredential, error) {

	client, err := civogo.NewClient(os.Getenv("CIVO_TOKEN"), region)
	if err != nil {
		log.Info().Msg(err.Error())
		return civogo.ObjectStoreCredential{}, err
	}

	creds, err := client.GetObjectStoreCredential(id)
	if err != nil {
		return civogo.ObjectStoreCredential{}, err
	}
	return *creds, nil
}
