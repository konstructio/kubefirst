package ssl

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cm "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/rs/zerolog/log"

	ghoddsYaml "github.com/ghodss/yaml"
	"github.com/kubefirst/kubefirst/configs"
	"github.com/kubefirst/kubefirst/internal/aws"
	"github.com/kubefirst/kubefirst/internal/k3d"
	"github.com/kubefirst/kubefirst/internal/k8s"
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/spf13/viper"
	yaml2 "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
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
			yamlObj, err := ghoddsYaml.JSONToYAML(jsonObj)
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

// CreateCertificatesForLocalWrapper groups a certification creation call into a wrapper. The provided application
// list is used to create SSL certificates for each of the provided application.
func CreateCertificatesForK3dWrapper(config k3d.K3dConfig) error {

	// create folder
	// todo: check permission
	err := os.Mkdir(config.MkCertPemDir, 0755)
	if err != nil && os.IsNotExist(err) {
		return err
	}

	for _, cert := range pkg.GetCertificateAppList() {
		if err := createCertificateForK3d(config, cert); err != nil {
			return err
		}
	}

	return nil
}

// createCertificateForLocal issue certificates for a specific application. MkCert is the tool who is going to create
// the certificates, store them in files, and store the certificates in the host trusted store.
func createCertificateForK3d(config k3d.K3dConfig, app pkg.CertificateAppList) error {

	fullAppAddress := app.AppName + "." + pkg.LocalDNS                    // example: app-name.localdev.me
	certFileName := config.MkCertPemDir + "/" + app.AppName + "-cert.pem" // example: app-name-cert.pem
	keyFileName := config.MkCertPemDir + "/" + app.AppName + "-key.pem"   // example: app-name-key.pem

	log.Info().Msgf("generating certificate %s.localdev.me on %s", app.AppName, config.MkCertClient)

	_, _, err := pkg.ExecShellReturnStrings(
		config.MkCertClient,
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

func CreateSecretsFromCertificatesForK3dWrapper(config *k3d.K3dConfig) error {

	for _, app := range pkg.GetCertificateAppList() {

		certFileName := config.MkCertPemDir + "/" + app.AppName + "-cert.pem" // example: app-name-cert.pem
		keyFileName := config.MkCertPemDir + "/" + app.AppName + "-key.pem"   // example: app-name-key.pem

		log.Info().Msgf("creating TLS k8s secret for %s...", app.AppName)

		// open file content
		certContent, err := pkg.GetFileContent(certFileName)
		if err != nil {
			return err
		}

		keyContent, err := pkg.GetFileContent(keyFileName)
		if err != nil {
			return err
		}

		data := make(map[string][]byte)
		data["tls.crt"] = certContent
		data["tls.key"] = keyContent

		// save content into secret
		err = k8s.CreateSecret(config.Kubeconfig, app.Namespace, app.AppName+"-tls", data) // todo argument 1 needs to be real
		if err != nil {
			log.Error().Err(err).Msgf("Error creating TLS k8s secret")
		}

		log.Info().Msgf("creating TLS k8s secret for %s done", app.AppName)
	}

	return nil
}

func Restore(backupDir, domainName, kubeconfigPath string) error {

	sslSecretFiles, err := ioutil.ReadDir(backupDir + "/secrets")
	if err != nil {
		return err
	}

	clientset, err := k8s.GetClientSet(false, kubeconfigPath)
	if err != nil {
		return err
	}

	for _, secret := range sslSecretFiles {

		// file is named with convention $namespace-$secretName.yaml
		//  todo link to backup source code
		namespace := strings.Split(secret.Name(), "-")[0]
		log.Info().Msg("creating secret: " + secret.Name())

		f, err := os.ReadFile(backupDir + "/secrets/" + secret.Name())
		if err != nil {
			return err
		}

		secretData := &v1.SecretApplyConfiguration{}

		err = yaml.Unmarshal(f, secretData)
		if err != nil {
			return err
		}

		sec, err := clientset.CoreV1().Secrets(namespace).Apply(context.Background(), secretData, metav1.ApplyOptions{FieldManager: "application/apply-patch"})
		if err != nil {
			return err
		}
		log.Info().Msgf("created secret: %s", sec.Name)
	}
	return nil
}

func Backup(backupDir, domainName, k1Dir, kubeconfigPath string) error {

	clientset, err := k8s.GetClientSet(false, kubeconfigPath)
	if err != nil {
		return err
	}

	//* corev1 secret resources
	secrets, err := clientset.CoreV1().Secrets("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		if strings.Contains(secret.Name, "-tls") {
			log.Info().Msg("backing up secret (ns/resource): " + secret.Namespace + "/" + secret.Name)

			// modify fields of secret for restore
			secret.APIVersion = "v1"
			secret.Kind = "Secret"
			secret.SetManagedFields(nil)
			secret.SetOwnerReferences(nil)
			secret.SetAnnotations(nil)
			secret.SetCreationTimestamp(metav1.Time{})
			secret.SetResourceVersion("")
			secret.SetUID("")

			fileName := fmt.Sprintf("%s/%s-%s.yaml", backupDir+"/secrets", secret.Namespace, secret.Name)
			log.Info().Msgf("writing file: %s\n\n", fileName)
			yamlContent, err := yaml.Marshal(secret)
			if err != nil {
				return fmt.Errorf("unable to marshal yaml: %s", err)
			}
			pkg.CreateFile(fileName, yamlContent)

		} else {
			log.Info().Msgf("skipping secret: %s", secret.Name)
		}
	}

	//* cert manager certificate resources
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return err
	}
	cmClientSet, err := cm.NewForConfig(k8sConfig)
	if err != nil {
		return err
	}

	clusterIssuers, err := cmClientSet.CertmanagerV1().ClusterIssuers().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, clusterissuer := range clusterIssuers.Items {
		clusterissuer.SetManagedFields(nil)
		clusterissuer.SetOwnerReferences(nil)
		clusterissuer.SetAnnotations(nil)
		clusterissuer.SetResourceVersion("")
		clusterissuer.SetCreationTimestamp(metav1.Time{})
		clusterissuer.SetUID("")
		clusterissuer.Status = cmv1.IssuerStatus{}

		fileName := fmt.Sprintf("%s/%s.yaml", backupDir+"/clusterissuers", clusterissuer.Name)
		log.Info().Msgf("writing file: %s\n", fileName)
		yamlContent, err := yaml.Marshal(clusterissuer)
		if err != nil {
			return fmt.Errorf("unable to marshal yaml: %s", err)
		}
		pkg.CreateFile(fileName, yamlContent)
	}

	certs, err := cmClientSet.CertmanagerV1().Certificates("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Info().Msg("error getting list of certificates")
	}

	for _, cert := range certs.Items {
		if strings.Contains(cert.Name, "-tls") {
			log.Info().Msg("backing up certificate (ns/resource): " + cert.Namespace + "/" + cert.Name)
			cert.SetManagedFields(nil)
			cert.SetOwnerReferences(nil)
			cert.SetAnnotations(nil)
			cert.SetResourceVersion("")
			cert.Status = cmv1.CertificateStatus{}
			cert.SetCreationTimestamp(metav1.Time{})
			cert.SetUID("")

			fileName := fmt.Sprintf("%s/%s-%s.yaml", backupDir+"/certificates", cert.Namespace, cert.Name)
			log.Info().Msgf("writing file: %s\n", fileName)
			yamlContent, err := yaml.Marshal(cert)
			if err != nil {
				return fmt.Errorf("unable to marshal yaml: %s", err)
			}
			pkg.CreateFile(fileName, yamlContent)
		} else {
			log.Info().Msg("skipping certficate (ns/resource): " + cert.Namespace + "/" + cert.Name)
		}
	}
	return nil
}
