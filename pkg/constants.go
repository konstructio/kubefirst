/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"fmt"
	"runtime"
)

var (
	BetaProviders []string = []string{"digitalocean", "vultr"}
)

const (
	JSONContentType              = "application/json"
	SoftServerURI                = "ssh://127.0.0.1:8022/config"
	GitHubOAuthClientId          = "2ced340927e0a6c49a45"
	CloudK3d                     = "k3d"
	CloudAws                     = "aws"
	DefaultS3Region              = "us-east-1"
	GitHubProviderName           = "github"
	GitHubHost                   = "github.com"
	LocalClusterName             = "kubefirst"
	MinimumAvailableDiskSize     = 10 // 10 GB
	KubefirstGitOpsRepository    = "gitops"
	KubefirstGitOpsRepositoryURL = "https://github.com/kubefirst/gitops-template"
	LocalDomainName              = "kubefirst.dev"
	LocalhostARCH                = runtime.GOARCH
	LocalhostOS                  = runtime.GOOS
	AwsECRUsername               = "AWS"
	RegistryAppName              = "registry"
	MinioDefaultUsername         = "k-ray"
	MinioDefaultPassword         = "feedkraystars"

	// github.com/kubefirst/manifests ref ver
	KubefirstManifestRepoRef = "0.1.0"
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
	VaultPortForwardURL = "http://localhost:8200"
)

var (
	VaultLocalURL    = fmt.Sprintf("http://vault.%s", LocalDomainName)
	VaultLocalURLTLS = fmt.Sprintf("https://vault.%s", LocalDomainName)
)

// Argo
const (
	ArgoPodName          = "argo-server"
	ArgoNamespace        = "argo"
	ArgoPodPort          = 2746
	ArgoPodLocalPort     = 2746
	ArgocdPortForwardURL = "http://localhost:8080"
)

var (
	ArgoLocalURLTLS = fmt.Sprintf("https://argo.%s", LocalDomainName)
)

// ArgoCD
const (
	ArgoCDPodName      = "argocd-server"
	ArgoCDNamespace    = "argocd"
	ArgoCDPodPort      = 8080
	ArgoCDPodLocalPort = 8080
	ArgoCDLocalBaseURL = "https://localhost:8080/api/v1"
)

var (
	ArgoCDLocalURL    = fmt.Sprintf("http://argocd.%s", LocalDomainName)
	ArgoCDLocalURLTLS = fmt.Sprintf("https://argocd.%s", LocalDomainName)
)

// ChartMuseum
const (
	ChartmuseumPodName      = "chartmuseum"
	ChartmuseumNamespace    = "chartmuseum"
	ChartmuseumPodPort      = 8080
	ChartmuseumPodLocalPort = 8181
	ChartmuseumLocalURL     = "http://localhost:8181"
)

var (
	ChartmuseumLocalURLTLS = fmt.Sprintf("https://chartmuseum.%s", LocalDomainName)
)

// Minio
const (
	MinioPodName             = "minio"
	MinioNamespace           = "minio"
	MinioPodPort             = 9000
	MinioPodLocalPort        = 9000
	MinioPortForwardEndpoint = "localhost:9000"
	MinioRegion              = "us-k3d-1"
)

var (
	MinioURL = fmt.Sprintf("https://minio.%s", LocalDomainName)
)

// Minio Console
const (
	MinioConsolePodName      = "minio"
	MinioConsoleNamespace    = "minio"
	MinioConsolePodPort      = 9001
	MinioConsolePodLocalPort = 9001
)

var (
	MinioConsoleURLTLS = fmt.Sprintf("https://minio-console.%s", LocalDomainName)
)

// Kubefirst Console
const (
	KubefirstConsolePodName       = "kubefirst-console"
	KubefirstConsoleNamespace     = "kubefirst"
	KubefirstConsolePodPort       = 80
	KubefirstConsolePodLocalPort  = 9094
	KubefirstConsoleLocalURLCloud = "http://localhost:9094"
)

var (
	KubefirstConsoleLocalURL    = fmt.Sprintf("http://kubefirst.%s", LocalDomainName)
	KubefirstConsoleLocalURLTLS = fmt.Sprintf("https://kubefirst.%s", LocalDomainName)
)

// Atlantis
const (
	AtlantisPodPort           = 4141
	AtlantisPodName           = "atlantis-0"
	AtlantisNamespace         = "atlantis"
	AtlantisPodLocalPort      = 4141
	LocalAtlantisURLTEMPORARY = "localhost:4141" // todo:
)

var (
	AtlantisLocalURLTEST = fmt.Sprintf("atlantis.%s", LocalDomainName)
	AtlantisLocalURL     = fmt.Sprintf("http://atlantis.%s", LocalDomainName)
	AtlantisLocalURLTLS  = fmt.Sprintf("https://atlantis.%s", LocalDomainName)
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

var (
	MetaphorFrontendSlimTLSDev     = fmt.Sprintf("https://metaphor-development.%s", LocalDomainName)
	MetaphorFrontendSlimTLSStaging = fmt.Sprintf("https://metaphor-staging.%s", LocalDomainName)
	MetaphorFrontendSlimTLSProd    = fmt.Sprintf("https://metaphor-production.%s", LocalDomainName)
)
