package pkg

const (
	ArgoCDLocalBaseURL  = "https://localhost:8080/api/v1"
	JSONContentType     = "application/json"
	SoftServerURI       = "ssh://127.0.0.1:8022/config"
	LocalAtlantisURL    = "localhost:4141"
	LocalConsoleUI      = "http://localhost:9094"
	GitHubOAuthClientId = "2ced340927e0a6c49a45"
	CloudK3d            = "k3d"
)

// SegmentIO constants
// SegmentIOWriteKey The write key is the unique identifier for a source that tells Segment which source data comes
// from, to which workspace the data belongs, and which destinations should receive the data.
const (
	SegmentIOWriteKey                 = "0gAYkX5RV3vt7s4pqCOOsDb6WHPLT30M"
	MetricInitStarted                 = "kubefirst.initialization.started"
	MetricInitCompleted               = "kubefirst.initialization.completed"
	MetricMgmtClusterInstallStarted   = "kubefirst.mgmt_cluster_install.started"
	MetricMgmtClusterInstallCompleted = "kubefirst.mgmt_cluster_install.completed"
)
