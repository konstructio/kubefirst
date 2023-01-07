package ssl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/ghodss/yaml"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	yaml2 "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

func getNamespacesToBackupSSL() (ns []string) {
	return []string{"argo", "argocd", "atlantis", "chartmuseum", "gitlab", "vault", "kubefirst"}
}

func getNSToBackupSSLMetaphorApps() (ns []string) {
	return []string{"staging", "development", "production"}
}

func getItemsToBackup(apiGroup string, apiVersion string, resourceType string, namespaces []string, jqQuery string) ([]string, error) {
	config := configs.ReadConfig()

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", config.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error getting k8sClient %s", err)
	}

	k8sClient := dynamic.NewForConfigOrDie(k8sConfig)

	//creating folder to store certificates backup, and continue if exists.
	if err := os.Mkdir(fmt.Sprintf("%s", config.CertsPath), os.ModePerm); err != nil {
		log.Warn().Msgf("error: could not create directory %q - it must exist to continue. error is: %s", config.CertsPath, err)
	}

	var files []string
	var items []unstructured.Unstructured
	for _, namespace := range namespaces {
		if len(jqQuery) > 0 {
			log.Info().Msg("getting resources and filtering using jq")
			items, err = k8s.GetResourcesByJq(k8sClient, context.TODO(), apiGroup, apiVersion, resourceType, namespace, jqQuery)
		} else {
			log.Info().Msg("getting resources")
			items, err = k8s.GetResourcesDynamically(k8sClient, context.TODO(), apiGroup, apiVersion, resourceType, namespace)
		}

		if err != nil {
			return nil, fmt.Errorf("error getting resources from k8s: %s", err)
		}

		for _, item := range items {
			jsonObj, err := item.MarshalJSON()
			if err != nil {
				return nil, fmt.Errorf("error converting object on json: %s", err)
			}
			//yamlObj, err := yaml.JSONToYAML(jsonObj)
			yamlObj, err := yaml.JSONToYAML(jsonObj)
			if err != nil {
				return nil, fmt.Errorf("error converting object from json to yaml: %s", err)
			}

			fileName := fmt.Sprintf("%s.%s", item.GetName(), "yaml")
			//TODO: test if kubeconfigpath is the correct value to write the files together another k1rst files
			fullFileName := filepath.Join(config.CertsPath, fileName)
			err = pkg.CreateFile(fullFileName, yamlObj)
			if err != nil {
				return nil, err
			}
			files = append(files, fullFileName)
		}
	}

	return files, nil
}

// GetBackupCertificates create a backup of Certificates on AWS S3 in yaml files
func GetBackupCertificates(includeMetaphorApps bool) (string, error) {
	log.Info().Msg("GetBackupCertificates called")
	awsProfile := viper.GetString("aws.profile")
	os.Setenv("AWS_PROFILE", awsProfile)
	bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
	aws.CreateBucket(false, bucketName)

	config := configs.ReadConfig()
	namespaces := getNamespacesToBackupSSL()

	if includeMetaphorApps {
		log.Info().Msg("Including Certificates from Metaphor Apps")
		namespaces = append(namespaces, getNSToBackupSSLMetaphorApps()...)
	}

	log.Info().Msg("getting certificates")
	certificates, err := getItemsToBackup("cert-manager.io", "v1", "certificates", namespaces, "")
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, cert := range certificates {
		fullPath := strings.Replace(cert, config.CertsPath, "/certs", 1)
		log.Info().Msg(fullPath)
		err = aws.UploadFile(bucketName, fullPath, cert)
		if err != nil {
			log.Info().Msg("there is an issue to uploaded your certificate to the S3 bucket")
			log.Panic().Msgf("%s", err)
		}
	}

	log.Info().Msg("getting secrets")
	query := ".metadata.annotations[\"cert-manager.io/issuer-kind\"] == \"ClusterIssuer\""
	secrets, err := getItemsToBackup("", "v1", "secrets", namespaces, query)
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, secret := range secrets {
		fullPath := strings.Replace(secret, config.CertsPath, "/secrets", 1)
		log.Info().Msg(fullPath)
		if err = aws.UploadFile(bucketName, fullPath, secret); err != nil {
			return "", err
		}
	}

	emptyNS := []string{""}
	log.Info().Msg("getting clusterissuers")
	clusterIssuers, err := getItemsToBackup("cert-manager.io", "v1", "clusterissuers", emptyNS, "")
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, clusterIssuer := range clusterIssuers {
		fullPath := strings.Replace(clusterIssuer, config.CertsPath, "/clusterissuers", 1)
		log.Info().Msg(fullPath)
		if err = aws.UploadFile(bucketName, fullPath, clusterIssuer); err != nil {
			return "", err
		}
	}

	return "Backup Cert-Manager resources finished successfully!", nil
}

// RestoreSSL - Restore Cluster certs from a previous install
func RestoreSSL(dryRun bool, includeMetaphorApps bool) error {
	config := configs.ReadConfig()

	if viper.GetBool("create.state.ssl.restored") {
		log.Info().Msg("Step already executed before, RestoreSSL skipped.")
		return nil
	}

	if dryRun {
		log.Info().Msg("[#99] Dry-run mode, RestoreSSL skipped.")
		return nil
	}
	namespaces := getNamespacesToBackupSSL()
	if includeMetaphorApps {
		log.Info().Msg("Including Certificates from Metaphor Apps")
		namespaces = append(namespaces, getNSToBackupSSLMetaphorApps()...)
	}
	for _, ns := range namespaces {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "create", "ns", ns)
		if err != nil {
			log.Info().Msgf("failed to create ns: %s, assuming that exists...", err)
		}
	}
	bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
	err := aws.DownloadBucket(bucketName, config.CertsPath)
	if err != nil {
		log.Info().Msgf("Error RestoreSSL: %s", err)
	}
	//! We need apply secrets firstly than other resources, accordingly with cert-manager docs
	//pathsRestored := []string{"secrets", "certs", "clusterissuers"}
	//! At this moment, we dont have the crds certs/clusterissuers installed on cluster
	pathsRestored := []string{"secrets"}
	for _, path := range pathsRestored {
		log.Info().Msg(path)
		//clean yaml
		//TODO filter yaml extension
		files, err := os.ReadDir(fmt.Sprintf("%s/%s", filepath.Join(config.CertsPath, path), "/"))
		if err != nil {
			log.Warn().Msgf("Error RestoreSSL: %s", err)
			return fmt.Errorf("erro: %s", err)
		}

		for _, f := range files {
			log.Info().Msg(f.Name())
			pathyaml := fmt.Sprintf("%s/%s", filepath.Join(config.CertsPath, path), f.Name())

			yfile, err := os.ReadFile(pathyaml)

			if err != nil {
				log.Info().Msgf("Error RestoreSSL: %s", err)
				return fmt.Errorf("erro: %s", err)
			}

			data := make(map[interface{}]interface{})

			err = yaml2.Unmarshal(yfile, &data)

			if err != nil {
				log.Info().Msgf("Error RestoreSSL: %s", err)
				return fmt.Errorf("erro: %s", err)
			}

			metadataMap := data["metadata"].(map[interface{}]interface{})
			delete(metadataMap, "resourceVersion")
			delete(metadataMap, "uid")
			delete(metadataMap, "creationTimestamp")
			delete(metadataMap, "managedFields")
			data["metadata"] = metadataMap

			dataCleaned, err := yaml2.Marshal(&data)

			if err != nil {
				log.Warn().Msgf("Error RestoreSSL: %s", err)
				return fmt.Errorf("erro: %s", err)
			}

			err = os.WriteFile(fmt.Sprintf("%s%s", pathyaml, ".clean"), dataCleaned, 0644)

			if err != nil {
				log.Warn().Msgf("Error RestoreSSL: %s", err)
				return fmt.Errorf("erro: %s", err)
			}

			log.Info().Msg("yaml cleaned written")
		}

		log.Info().Msgf("applying the folder: %s", path)
		_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "apply", "-f", filepath.Join(config.CertsPath, path))
		if err != nil {
			log.Warn().Msgf("failed to apply %s: %s, assuming that exists...", path, err)
		}
	}
	viper.Set("create.state.ssl.restored", true)
	viper.WriteConfig()
	return nil
}

func InstallCALocal(config *configs.Config) {
	_, _, err := pkg.ExecShellReturnStrings(config.MkCertPath, "-install")
	if err != nil {
		log.Warn().Msgf("failed to uninstall CA of mkCert: %s", err)
	}
}

// todo: make destroy call it
func UninstallCALocal(config *configs.Config) {
	_, _, err := pkg.ExecShellReturnStrings(config.MkCertPath, "-uninstall")
	if err != nil {
		log.Warn().Msgf("failed to uninstall CA of mkCert: %s", err)
	}
}

// CreateCertificatesForLocalWrapper groups a certification creation call into a wrapper. The provided application
// list is used to create SSL certificates for each of the provided application.
func CreateCertificatesForLocalWrapper(config *configs.Config) error {

	// create folder
	// todo: check permission
	err := os.Mkdir(config.MkCertPemFilesPath, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	for _, cert := range pkg.GetCertificateAppList() {
		if err := createCertificateForLocal(config, cert); err != nil {
			return err
		}
	}

	return nil
}

// createCertificateForLocal issue certificates for a specific application. MkCert is the tool who is going to create
// the certificates, store them in files, and store the certificates in the host trusted store.
func createCertificateForLocal(config *configs.Config, app pkg.CertificateAppList) error {

	fullAppAddress := app.AppName + "." + pkg.LocalDNS                    // example: app-name.localdev.me
	certFileName := config.MkCertPemFilesPath + app.AppName + "-cert.pem" // example: app-name-cert.pem
	keyFileName := config.MkCertPemFilesPath + app.AppName + "-key.pem"   // example: app-name-key.pem

	log.Info().Msgf("generating certificate %s.localdev.me on %s", app.AppName, config.MkCertPath)

	_, _, err := pkg.ExecShellReturnStrings(
		config.MkCertPath,
		"-cert-file",
		certFileName,
		"-key-file",
		keyFileName,
		pkg.LocalDNS,
		fullAppAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to generate %s SSL certificate using MkCert: %v", app.AppName, err)
	}

	return nil
}
