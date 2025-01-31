package provision

import (
	"testing"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockClusterClient struct {
	clusters map[string]apiTypes.Cluster
}

func (m *MockClusterClient) GetCluster(clusterName string) (*apiTypes.Cluster, error) {
	foundCluster, exists := m.clusters[clusterName]
	if !exists {
		return nil, cluster.ErrNotFound
	}
	return &foundCluster, nil
}

func (m *MockClusterClient) CreateCluster(cluster apiTypes.ClusterDefinition) error {
	return nil
}

func (m *MockClusterClient) ResetClusterProgress(clusterName string) error {
	return nil
}

func TestClusterProvision(t *testing.T) {
	t.Run("should have checks after initialized", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		assert.Equal(t, "test-cluster", cp.clusterName)
		assert.Equal(t, InstallToolsCheck, cp.installSteps[0].StepName)
	})

	t.Run("should have InstallTools check first", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		require.Greater(t, len(cp.installSteps), 0)
		assert.Equal(t, InstallToolsCheck, cp.installSteps[0].StepName)
	})

	t.Run("should be complete if there are no more install steps", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		assert.False(t, cp.IsComplete())

		for range cp.installSteps {
			cp.popStep()
		}

		assert.True(t, cp.IsComplete())
	})

	t.Run("should continue to show complete if attempting to update the process", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		for range cp.installSteps {
			cp.popStep()
		}

		step := cp.popStep()
		assert.Equal(t, ProvisionComplete, step)
	})

	t.Run("should move to the next step when popped", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		assert.Equal(t, InstallToolsCheck, cp.GetCurrentStep())
		cp.popStep()
		assert.Equal(t, DomainLivenessCheck, cp.GetCurrentStep())
	})

	t.Run("should change the current step if the top is popped", func(t *testing.T) {
		client := &MockClusterClient{}
		cp := NewProvisionWatcher("test-cluster", client)

		step := cp.popStep()
		assert.Equal(t, InstallToolsCheck, step)
		assert.Equal(t, DomainLivenessCheck, cp.GetCurrentStep())
	})

	t.Run("should only update one step at a time even if all checks are complete", func(t *testing.T) {
		client := &MockClusterClient{
			clusters: map[string]apiTypes.Cluster{
				"test-cluster": {
					ClusterName:                "test-cluster",
					InstallToolsCheck:          true,
					DomainLivenessCheck:        true,
					KbotSetupCheck:             true,
					GitInitCheck:               true,
					GitopsReadyCheck:           true,
					GitTerraformApplyCheck:     true,
					GitopsPushedCheck:          true,
					CloudTerraformApplyCheck:   true,
					ClusterSecretsCreatedCheck: true,
					ArgoCDInstallCheck:         true,
					ArgoCDInitializeCheck:      true,
					VaultInitializedCheck:      true,
					VaultTerraformApplyCheck:   true,
					UsersTerraformApplyCheck:   true,
				},
			},
		}
		cp := NewProvisionWatcher("test-cluster", client)

		err := cp.UpdateProvisionProgress()
		assert.NoError(t, err)
		assert.Equal(t, DomainLivenessCheck, cp.GetCurrentStep())
	})

	t.Run("should return an error if the cluster is in an error state", func(t *testing.T) {
		client := &MockClusterClient{
			clusters: map[string]apiTypes.Cluster{
				"test-cluster": {
					ClusterName:   "test-cluster",
					Status:        "error",
					LastCondition: "some error",
				},
			},
		}
		cp := NewProvisionWatcher("test-cluster", client)

		err := cp.UpdateProvisionProgress()
		assert.Error(t, err)
	})

	t.Run("should not return an error if the cluster isn't ready", func(t *testing.T) {
		client := &MockClusterClient{
			clusters: map[string]apiTypes.Cluster{},
		}
		cp := NewProvisionWatcher("test-cluster", client)

		err := cp.UpdateProvisionProgress()
		assert.NoError(t, err)
	})
}
