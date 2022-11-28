package pkg

const (
	JSONContentType          = "application/json"
	SoftServerURI            = "ssh://127.0.0.1:8022/config"
	GitHubOAuthClientId      = "2ced340927e0a6c49a45"
	CloudK3d                 = "k3d"
	GitHubProviderName       = "github"
	GitHubHost               = "github.com"
	LocalClusterName         = "kubefirst"
	MinimumAvailableDiskSize = 10 // 10 GB
	LocalDNS                 = "localdev.me"
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

// Helm
const (
	HelmRepoName         = "argo"
	HelmRepoURL          = "https://argoproj.github.io/argo-helm"
	HelmRepoChartName    = "argo-cd"
	HelmRepoNamespace    = "argocd"
	HelmRepoChartVersion = "4.10.5"
)

// Vault
const (
	VaultPodName      = "vault-0"
	VaultNamespace    = "vault"
	VaultPodPort      = 8200
	VaultPodLocalPort = 8200
	VaultLocalURL     = "http://vault.localdev.me"
	VaultLocalURLTLS  = "https://vault.localdev.me"
)

// Argo
const (
	ArgoPodName      = "argo-server"
	ArgoNamespace    = "argo"
	ArgoPodPort      = 2746
	ArgoPodLocalPort = 2746
	ArgoLocalURLTLS  = "https://argo.localdev.me"
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
	ChartmuseumLocalURL     = "http://chartmuseum.localdev.me"
	ChartmuseumLocalURLTLS  = "https://chartmuseum.localdev.me"
)

// Minio
const (
	MinioPodName      = "minio"
	MinioNamespace    = "minio"
	MinioPodPort      = 9000
	MinioPodLocalPort = 9000
	MinioURL          = "http://minio.localdev.me"
	MinioURLTLS       = "https://minio.localdev.me"
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
	KubefirstConsoleLocalURL      = "http://kubefirst-console.localdev.me"
	KubefirstConsoleLocalURLTLS   = "https://kubefirst-console.localdev.me"
)

// Atlantis
const (
	AtlantisPodName           = "atlantis-0"
	AtlantisNamespace         = "atlantis"
	AtlantisPodPort           = 4141
	AtlantisPodLocalPort      = 4141
	AtlantisLocalURLTEST      = "atlantis.localdev.me"
	AtlantisLocalURL          = "http://atlantis.localdev.me"
	AtlantisLocalURLTLS       = "https://atlantis.localdev.me"
	LocalAtlantisURLTEMPORARY = "localhost:4141" // todo:
	//LocalAtlantisURL = "atlantis.localdev.me" // todo:
)
