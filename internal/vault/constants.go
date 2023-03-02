package vault

const (
	// Rate at which to check for Vault health
	checkInterval int = 10
	// Vault API endpoints
	vaultHealthEndpoint string = "/v1/sys/health"
	vaultInitEndpoint   string = "/v1/sys/init"
	vaultRaftEndpoint   string = "/sys/storage/raft"
	vaultUnsealEndpoint string = "/v1/sys/unseal"
	// For raft, vault-0 will always be primary referenced by its endpoint
	vaultRaftPrimaryAddress string = "http://vault-0.vault-internal"
	// Name for the Secret that gets created that contains root auth data
	VaultSecretName string = "vault-unseal-secret"
	// Namespace that Vault runs in
	VaultNamespace string = "vault"
	// number of recovery shares for Vault unseal
	RecoveryShares int = 5
	// number of recovery keys for Vault
	RecoveryThreshold int = 3
	// number of secret shares for Vault unseal
	SecretShares = 5
	// number of secret threshold Vault unseal
	SecretThreshold = 3
)
