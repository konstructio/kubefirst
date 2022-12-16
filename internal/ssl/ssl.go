package ssl

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	return []string{"argo", "argocd", "atlantis", "chartmuseum", "gitlab", "vault"}
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
		log.Printf("error: could not create directory %q - it must exist to continue. error is: %s", config.CertsPath, err)
	}

	var files []string
	var items []unstructured.Unstructured
	for _, namespace := range namespaces {
		if len(jqQuery) > 0 {
			log.Println("getting resources and filtering using jq")
			items, err = k8s.GetResourcesByJq(k8sClient, context.TODO(), apiGroup, apiVersion, resourceType, namespace, jqQuery)
		} else {
			log.Println("getting resources")
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
	log.Println("GetBackupCertificates called")
	awsProfile := viper.GetString("aws.profile")
	os.Setenv("AWS_PROFILE", awsProfile)
	bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
	aws.CreateBucket(false, bucketName)

	config := configs.ReadConfig()
	namespaces := getNamespacesToBackupSSL()

	if includeMetaphorApps {
		log.Println("Including Certificates from Metaphor Apps")
		namespaces = append(namespaces, getNSToBackupSSLMetaphorApps()...)
	}

	log.Println("getting certificates")
	certificates, err := getItemsToBackup("cert-manager.io", "v1", "certificates", namespaces, "")
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, cert := range certificates {
		fullPath := strings.Replace(cert, config.CertsPath, "/certs", 1)
		log.Println(fullPath)
		err = aws.UploadFile(bucketName, fullPath, cert)
		if err != nil {
			log.Println("there is an issue to uploaded your certificate to the S3 bucket")
			log.Panic(err)
		}
	}

	log.Println("getting secrets")
	query := ".metadata.annotations[\"cert-manager.io/issuer-kind\"] == \"ClusterIssuer\""
	secrets, err := getItemsToBackup("", "v1", "secrets", namespaces, query)
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, secret := range secrets {
		fullPath := strings.Replace(secret, config.CertsPath, "/secrets", 1)
		log.Println(fullPath)
		if err = aws.UploadFile(bucketName, fullPath, secret); err != nil {
			return "", err
		}
	}

	emptyNS := []string{""}
	log.Println("getting clusterissuers")
	clusterIssuers, err := getItemsToBackup("cert-manager.io", "v1", "clusterissuers", emptyNS, "")
	if err != nil {
		return "", fmt.Errorf("erro: %s", err)
	}
	for _, clusterIssuer := range clusterIssuers {
		fullPath := strings.Replace(clusterIssuer, config.CertsPath, "/clusterissuers", 1)
		log.Println(fullPath)
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
		log.Printf("Step already executed before, RestoreSSL skipped.")
		return nil
	}

	if dryRun {
		log.Printf("[#99] Dry-run mode, RestoreSSL skipped.")
		return nil
	}
	namespaces := getNamespacesToBackupSSL()
	if includeMetaphorApps {
		log.Println("Including Certificates from Metaphor Apps")
		namespaces = append(namespaces, getNSToBackupSSLMetaphorApps()...)
	}
	for _, ns := range namespaces {
		_, _, err := pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "create", "ns", ns)
		if err != nil {
			log.Printf("failed to create ns: %s, assuming that exists...", err)
		}
	}
	bucketName := fmt.Sprintf("k1-%s", viper.GetString("aws.hostedzonename"))
	err := aws.DownloadBucket(bucketName, config.CertsPath)
	if err != nil {
		log.Println("Error RestoreSSL:", err)
	}
	//! We need apply secrets firstly than other resources, accordingly with cert-manager docs
	//pathsRestored := []string{"secrets", "certs", "clusterissuers"}
	//! At this moment, we dont have the crds certs/clusterissuers installed on cluster
	pathsRestored := []string{"secrets"}
	for _, path := range pathsRestored {
		log.Print(path)
		//clean yaml
		//TODO filter yaml extension
		files, err := os.ReadDir(fmt.Sprintf("%s/%s", filepath.Join(config.CertsPath, path), "/"))
		if err != nil {
			log.Println("Error RestoreSSL:", err)
			return fmt.Errorf("erro: %s", err)
		}

		for _, f := range files {
			log.Println(f.Name())
			pathyaml := fmt.Sprintf("%s/%s", filepath.Join(config.CertsPath, path), f.Name())

			yfile, err := os.ReadFile(pathyaml)

			if err != nil {
				log.Println("Error RestoreSSL:", err)
				return fmt.Errorf("erro: %s", err)
			}

			data := make(map[interface{}]interface{})

			err = yaml2.Unmarshal(yfile, &data)

			if err != nil {
				log.Println("Error RestoreSSL:", err)
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
				log.Println("Error RestoreSSL:", err)
				return fmt.Errorf("erro: %s", err)
			}

			err = os.WriteFile(fmt.Sprintf("%s%s", pathyaml, ".clean"), dataCleaned, 0644)

			if err != nil {
				log.Println("Error RestoreSSL:", err)
				return fmt.Errorf("erro: %s", err)
			}

			log.Println("yaml cleaned written")
		}

		log.Printf("applying the folder: %s", path)
		_, _, err = pkg.ExecShellReturnStrings(config.KubectlClientPath, "--kubeconfig", config.KubeConfigPath, "apply", "-f", filepath.Join(config.CertsPath, path))
		if err != nil {
			log.Printf("failed to apply %s: %s, assuming that exists...", path, err)
		}
	}
	viper.Set("create.state.ssl.restored", true)
	viper.WriteConfig()
	return nil
}

// InstallMKCertLocal install mkCert state store in the host machine
// todo: update logs
// todo: holding this code until next release (1.12),
// if this code is still here after 1.12, please feel free to delete
func InstallMKCertLocal(config *configs.Config, disableTLS bool) error {

	if disableTLS {
		log.Println(pkg.TLSSupportIsDisabled)
		return nil
	}

	log.Println("installing CA from MkCert")

	_, output, err := pkg.ExecShellReturnStrings(config.MkCertPath, "-install")
	if err != nil {
		return fmt.Errorf("failed to install certificates using mkCert, you won't have valid certificates "+
			"for the provisioned services. error: %v, output: %v", err, output)
	}

	log.Println(output)

	if strings.Contains(output, "The local CA is already installed in the system trust store!") {
		log.Println("mkCert already installed")
		return nil
	}

	if strings.Contains(output, "The local CA is now installed in the system trust store!") {
		log.Println("mkCert was successfully installed")
		return nil
	}

	return fmt.Errorf("failed to install certificates using mkCert, that means you won't have valid "+
		"certificates for the provisioned services. error: %v, output: %v", err, output,
	)
}

// todo: make destroy call it
func UninstallCALocal(config *configs.Config) {
	_, _, err := pkg.ExecShellReturnStrings(config.MkCertPath, "-uninstall")
	if err != nil {
		log.Printf("failed to uninstall CA of mkCert: %s", err)
	}
}

// CreateCertificatesForLocalWrapper groups a certification creation call into a wrapper. The provided application
// list is used to create SSL certificates for each of the provided application.
func CreateCertificatesForLocalWrapper(config *configs.Config) error {

	log.Println("creating local certificates")

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

	log.Println("creating local certificates")

	return nil
}

// createCertificateForLocal issue certificates for a specific application. MkCert is the tool who is going to create
// the certificates, store them in files, and store the certificates in the host trusted store.
func createCertificateForLocal(config *configs.Config, app pkg.CertificateAppList) error {

	fullAppAddress := app.AppName + "." + pkg.LocalDNS                    // example: app-name.localdev.me
	certFileName := config.MkCertPemFilesPath + app.AppName + "-cert.pem" // example: app-name-cert.pem
	keyFileName := config.MkCertPemFilesPath + app.AppName + "-key.pem"   // example: app-name-key.pem

	log.Printf("generating certificate %s.localdev.me on %s", app.AppName, config.MkCertPath)

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
