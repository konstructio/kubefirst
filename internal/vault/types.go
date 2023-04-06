/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package vault

import vaultapi "github.com/hashicorp/vault/api"

// HealthResponse specifies the content of a health response from a vault API
// https://developer.hashicorp.com/vault/api-docs/system/health#sample-response
type HealthResponse struct {
	Initialized                bool   `json:"initialized"`
	Sealed                     bool   `json:"sealed"`
	Standby                    bool   `json:"standby"`
	PerformanceStandby         bool   `json:"performance_standby"`
	ReplicationPerformanceMode string `json:"replication_performance_mode"`
	ReplicationDRMode          string `json:"replication_dr_mode"`
	ServerTimeUTC              int    `json:"server_time_utc"`
	Version                    string `json:"version"`
	ClusterName                string `json:"cluster_name"`
	ClusterID                  string `json:"cluster_id"`
}

// InitRequest specifies the content of an `init` operation against a vault API
// https://developer.hashicorp.com/vault/api-docs/system/init#sample-payload
type InitRequest struct {
	SecretShares    int `json:"secret_shares"`
	SecretThreshold int `json:"secret_threshold"`
}

// InitResponse specifies the content of an `init` operation response from a vault API
// https://developer.hashicorp.com/vault/api-docs/system/init#sample-response-1
type InitResponse struct {
	Keys       []string `json:"keys"`
	KeysBase64 []string `json:"keys_base64"`
	RootToken  string   `json:"root_token"`
}

type RaftJoinRequest struct {
	LeaderAPIAddress string `json:"leader_api_addr"`
}

type RaftJoinResponse struct {
}

// UnsealRequest specifies the content of an `unseal` operation against a vault API
// https://developer.hashicorp.com/vault/api-docs/system/unseal#sample-payload
type UnsealRequest struct {
	Key string `json:"key"`
}

// UnsealResponse specifies the content of an `unseal` operation response from a vault API
// t holds the threshold and n holds the number of shares
// https://developer.hashicorp.com/vault/api-docs/system/unseal#sample-response
type UnsealResponse struct {
	Sealed      bool   `json:"sealed"`
	T           int    `json:"t"`
	N           int    `json:"n"`
	Progress    int    `json:"progress"`
	Version     string `json:"version"`
	ClusterName string `json:"cluster_name"`
	ClusterID   string `json:"cluster_id"`
}

type VaultUnsealOptions struct {
	HighAvailability     bool
	HighAvailabilityType string
	Nodes                int
	RaftLeader           bool
	RaftFollower         bool
	UseAPI               bool
	VaultAPIAddress      string
}

type VaultConfiguration struct {
	Config vaultapi.Config
}
