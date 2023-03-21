package pkg

import "runtime"

const (
	JSONContentType              = "application/json"
	SoftServerURI                = "ssh://127.0.0.1:8022/config"
	GitHubOAuthClientId          = "2ced340927e0a6c49a45"
	CloudK3d                     = "k3d"
	CloudAws                     = "aws"
	GitHubProviderName           = "github"
	GitHubHost                   = "github.com"
	LocalClusterName             = "kubefirst"
	MinimumAvailableDiskSize     = 10 // 10 GB
	KubefirstGitOpsRepository    = "gitops"
	KubefirstGitOpsRepositoryURL = "https://github.com/kubefirst/gitops-template"
	LocalDNS                     = "localdev.me"
	LocalhostARCH                = runtime.GOARCH
	LocalhostOS                  = runtime.GOOS
	AwsECRUsername               = "AWS"
	RegistryAppName              = "registry"
	MinioDefaultUsername         = "k-ray"
	MinioDefaultPassword         = "feedkraystars"
)

// SegmentIO constants
// SegmentIOWriteKey The write key is the unique identifier for a source that tells Segment which source data comes
// from, to which workspace the data belongs, and which destinations should receive the data.
const (
	SegmentIOWriteKey                 = "0gAYkX5RV3vt7s4pqCOOsDb6WHPLT30M"
	MetricInitStarted                 = "kubefirst.init.started"
	MetricInitCompleted               = "kubefirst.init.completed"
	MetricMgmtClusterInstallStarted   = "kubefirst.mgmt_cluster_install.started"
	MetricMgmtClusterInstallCompleted = "kubefirst.mgmt_cluster_install.completed"
)

// Vault
const (
	VaultPodName        = "vault-0"
	VaultNamespace      = "vault"
	VaultPodPort        = 8200
	VaultPodLocalPort   = 8200
	VaultLocalURL       = "http://vault.localdev.me"
	VaultLocalURLTLS    = "https://vault.localdev.me"
	VaultPortForwardURL = "http://localhost:8200"
)

// Argo
const (
	ArgoPodName          = "argo-server"
	ArgoNamespace        = "argo"
	ArgoPodPort          = 2746
	ArgoPodLocalPort     = 2746
	ArgoLocalURLTLS      = "https://argo.localdev.me"
	ArgocdPortForwardURL = "http://localhost:8080"
)

// ArgoCD
const (
	ArgoCDPodName      = "argocd-server"
	ArgoCDNamespace    = "argocd"
	ArgoCDPodPort      = 8080
	ArgoCDPodLocalPort = 8080
	ArgoCDLocalURL     = "http://argocd.localdev.me"
	ArgoCDLocalURLTLS  = "https://argocd.localdev.me"
	ArgoCDLocalBaseURL = "https://localhost:8080/api/v1"
)

// ChartMuseum
const (
	ChartmuseumPodName      = "chartmuseum"
	ChartmuseumNamespace    = "chartmuseum"
	ChartmuseumPodPort      = 8080
	ChartmuseumPodLocalPort = 8181
	ChartmuseumLocalURL     = "http://localhost:8181"
	ChartmuseumLocalURLTLS  = "https://chartmuseum.localdev.me"
)

// Minio
const (
	MinioPodName             = "minio"
	MinioNamespace           = "minio"
	MinioPodPort             = 9000
	MinioPodLocalPort        = 9000
	MinioURL                 = "https://minio.localdev.me"
	MinioPortForwardEndpoint = "localhost:9000"
	MinioRegion              = "us-k3d-1"
)

// Minio Console
const (
	MinioConsolePodName      = "minio"
	MinioConsoleNamespace    = "minio"
	MinioConsolePodPort      = 9001
	MinioConsolePodLocalPort = 9001
	MinioConsoleURLTLS       = "https://minio-console.localdev.me"
)

// Kubefirst Console
const (
	KubefirstConsolePodName       = "kubefirst-console"
	KubefirstConsoleNamespace     = "kubefirst"
	KubefirstConsolePodPort       = 80
	KubefirstConsolePodLocalPort  = 9094
	KubefirstConsoleLocalURLCloud = "http://localhost:9094"
	KubefirstConsoleLocalURL      = "http://kubefirst.localdev.me"
	KubefirstConsoleLocalURLTLS   = "https://kubefirst.localdev.me"
)

// Atlantis
const (
	AtlantisPodPort           = 4141
	AtlantisPodName           = "atlantis-0"
	AtlantisNamespace         = "atlantis"
	AtlantisPodLocalPort      = 4141
	AtlantisLocalURLTEST      = "atlantis.localdev.me"
	AtlantisLocalURL          = "http://atlantis.localdev.me"
	AtlantisLocalURLTLS       = "https://atlantis.localdev.me"
	LocalAtlantisURLTEMPORARY = "localhost:4141" // todo:
	//LocalAtlantisURL = "atlantis.localdev.me" // todo:
)

// MetaphorFrontendDevelopment
const (
	MetaphorFrontendDevelopmentServiceName      = "metaphor-development"
	MetaphorFrontendDevelopmentNamespace        = "development"
	MetaphorFrontendDevelopmentServicePort      = 443
	MetaphorFrontendDevelopmentServiceLocalPort = 4000
	MetaphorFrontendDevelopmentLocalURL         = "http://localhost:4000"
)

// MetaphorGoDevelopment
const (
	MetaphorGoDevelopmentServiceName      = "metaphor-go-development"
	MetaphorGoDevelopmentNamespace        = "development"
	MetaphorGoDevelopmentServicePort      = 443
	MetaphorGoDevelopmentServiceLocalPort = 5000
	MetaphorGoDevelopmentLocalURL         = "http://localhost:5000"
)

// MetaphorDevelopment
const (
	MetaphorDevelopmentServiceName      = "metaphor-development"
	MetaphorDevelopmentNamespace        = "development"
	MetaphorDevelopmentServicePort      = 443
	MetaphorDevelopmentServiceLocalPort = 3000
	MetaphorDevelopmentLocalURL         = "http://localhost:3000"
)

const (
	MetaphorFrontendSlimTLSDev     = "https://metaphor-development.localdev.me"
	MetaphorFrontendSlimTLSStaging = "https://metaphor-staging.localdev.me"
	MetaphorFrontendSlimTLSProd    = "https://metaphor-production.localdev.me"
)
