package provision

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/konstructio/kubefirst-api/pkg/types"
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
)

func TestPopStep(t *testing.T) {
	cp := NewClusterProvision("test-cluster")
	initialLength := len(cp.installSteps)

	// Test first pop
	firstStep := cp.installSteps[0].StepName
	poppedStep := cp.PopStep()

	if poppedStep != firstStep {
		t.Errorf("Expected popped step to be %s, got %s", firstStep, poppedStep)
	}

	if len(cp.installSteps) != initialLength-1 {
		t.Errorf("Expected length after pop to be %d, got %d", initialLength-1, len(cp.installSteps))
	}

	// Test that next step is now first
	if cp.installSteps[0].StepName != "Domain Liveness" {
		t.Errorf("Expected new first step to be 'Domain Liveness', got %s", cp.installSteps[0].StepName)
	}

	// Pop all remaining steps
	for i := 0; i < initialLength-1; i++ {
		cp.PopStep()
	}

}

func TestUpdateProvisionProgress(t *testing.T) {
	t.Run("client returns error", func(t *testing.T) {
		mockClient := &mockClusterClient{
			fnGetCluster: func(clusterName string) (*types.Cluster, error) {
				return nil, errors.New("error getting cluster")
			},
		}

		cp := NewClusterProvision("test-cluster")
		cp.client = mockClient

		err := cp.UpdateProvisionProgress()
		if err == nil {
			t.Error("expected error but got none")
		}
		if !strings.Contains(err.Error(), "error getting cluster") {
			t.Errorf("expected error containing 'error getting cluster', got %v", err)
		}
	})

	t.Run("cluster in error state", func(t *testing.T) {
		mockClient := &mockClusterClient{
			fnGetCluster: func(clusterName string) (*types.Cluster, error) {
				return &types.Cluster{
					Status:        "error",
					LastCondition: "cluster in error state",
				}, nil
			},
		}

		cp := NewClusterProvision("test-cluster")
		cp.client = mockClient

		err := cp.UpdateProvisionProgress()
		if err == nil {
			t.Error("expected error but got none")
		}
		if !strings.Contains(err.Error(), "cluster in error state") {
			t.Errorf("expected error containing 'cluster in error state', got %v", err)
		}
	})

	t.Run("successful progress update - step complete", func(t *testing.T) {
		mockClient := &mockClusterClient{
			fnGetCluster: func(clusterName string) (*types.Cluster, error) {
				return &types.Cluster{
					Status:            "running",
					InstallToolsCheck: true,
				}, nil
			},
		}

		cp := NewClusterProvision("test-cluster")
		cp.client = mockClient

		err := cp.UpdateProvisionProgress()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cp.GetCurrentStep() == InstallToolsCheck {
			t.Error("step should have been popped but wasn't")
		}
	})

	t.Run("successful progress update - step not complete", func(t *testing.T) {
		mockClient := &mockClusterClient{
			fnGetCluster: func(clusterName string) (*types.Cluster, error) {
				return &types.Cluster{
					Status:            "running",
					InstallToolsCheck: false,
				}, nil
			},
		}

		cp := NewClusterProvision("test-cluster")
		cp.client = mockClient

		err := cp.UpdateProvisionProgress()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

type mockClusterClient struct {
	fnGetCluster           func(clusterName string) (*types.Cluster, error)
	fnCreateCluster        func(cluster apiTypes.ClusterDefinition) error
	fnResetClusterProgress func(clusterName string) error
}

func (m *mockClusterClient) GetCluster(clusterName string) (*types.Cluster, error) {
	if m.fnGetCluster != nil {
		return m.fnGetCluster(clusterName)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockClusterClient) CreateCluster(cluster apiTypes.ClusterDefinition) error {
	if m.fnCreateCluster != nil {
		return m.fnCreateCluster(cluster)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockClusterClient) ResetClusterProgress(clusterName string) error {
	if m.fnResetClusterProgress != nil {
		return m.fnResetClusterProgress(clusterName)
	}
	return fmt.Errorf("not implemented")
}
