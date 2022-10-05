package pkg

const (
	ArgoCDLocalBaseURL = "https://localhost:8080/api/v1"
	JSONContentType    = "application/json"
	SoftServerURI      = "ssh://127.0.0.1:8022/config"
)

// SegmentIO constants
// SegmentIOWriteKey The write key is the unique identifier for a source that tells Segment which source data comes
// from, to which workspace the data belongs, and which destinations should receive the data.
const (
	SegmentIOWriteKey                 = "ZAeVUYRkV8TTKSuRlTeGzDhs5owGc84t"
	MetricInitStarted                 = "kubefirst.init.started"
	MetricInitCompleted               = "kubefirst.init.completed"
	MetricMgmtClusterInstallStarted   = "kubefirst.mgmt_cluster_install.started"
	MetricMgmtClusterInstallCompleted = "kubefirst.mgmt_cluster_install.completed"
)
