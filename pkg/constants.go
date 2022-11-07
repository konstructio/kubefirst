package pkg

const (
	ArgoCDLocalBaseURL       = "https://localhost:8080/api/v1"
	JSONContentType          = "application/json"
	SoftServerURI            = "ssh://127.0.0.1:8022/config"
	LocalAtlantisURL         = "localhost:4141"
	ConsoleUILocalURL        = "http://localhost:9094"
	ChartmuseumLocalURL      = "http://localhost:8181"
	ArgoLocalURL             = "http://localhost:2746"
	ArgoCDLocalURL           = "http://localhost:8080"
	VaultLocalURL            = "http://localhost:8200"
	AtlantisLocalURL         = "http://localhost:4141"
	MinioURL                 = "http://localhost:9000"
	MinioConsoleURL          = "http://localhost:9001"
	GitHubOAuthClientId      = "2ced340927e0a6c49a45"
	CloudK3d                 = "k3d"
	GitHubProviderName       = "github"
	GitHubHost               = "github.com"
	LocalClusterName         = "kubefirst"
	MinimumAvailableDiskSize = 10 // 10 GB
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

const (
	HelmRepoName         = "argo"
	HelmRepoURL          = "https://argoproj.github.io/argo-helm"
	HelmRepoChartName    = "argo-cd"
	HelmRepoNamespace    = "argocd"
	HelmRepoChartVersion = "4.10.5"
)
