package utilities

import (
	"fmt"
	"os"
	"testing"
)

func TestCreateK1ClusterDirectoryE(t *testing.T) {
	tests := []struct {
		name        string
		homePath    string
		clusterName string
		wantOk      bool
		wantErr     bool
	}{
		{
			name:        "successfully creates new directory",
			homePath:    t.TempDir(),
			clusterName: "test-cluster",
			wantOk:      true,
			wantErr:     false,
		},
		{
			name:        "empty cluster name",
			homePath:    t.TempDir(),
			clusterName: "",
			wantOk:      true,
			wantErr:     false,
		},
		{
			name:        "invalid home path",
			homePath:    "/nonexistent/path",
			clusterName: "test-cluster",
			wantOk:      false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateK1ClusterDirectoryE(tt.homePath, tt.clusterName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateK1ClusterDirectoryE() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				expectedPath := fmt.Sprintf("%s/.k1/%s", tt.homePath, tt.clusterName)
				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("Directory was not created at %s", expectedPath)
				}
			}
		})
	}
}
