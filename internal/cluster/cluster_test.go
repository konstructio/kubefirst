package cluster

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
)

type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestClusterClient_CreateCluster(t *testing.T) {
	tests := []struct {
		name        string
		cluster     apiTypes.ClusterDefinition
		mockResp    *http.Response
		mockErr     error
		wantErr     bool
		errContains string
	}{
		{
			name: "successful cluster creation",
			cluster: apiTypes.ClusterDefinition{
				ClusterName: "test-cluster",
			},
			mockResp: &http.Response{
				StatusCode: http.StatusAccepted,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"accepted"}`))),
			},
			wantErr: false,
		},
		{
			name: "error - unexpected status code",
			cluster: apiTypes.ClusterDefinition{
				ClusterName: "test-cluster",
			},
			mockResp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"bad request"}`))),
			},
			wantErr:     true,
			errContains: "unexpected status code",
		},
		{
			name: "error - failed to make request",
			cluster: apiTypes.ClusterDefinition{
				ClusterName: "test-cluster",
			},
			mockErr:     http.ErrHandlerTimeout,
			wantErr:     true,
			errContains: "failed to execute request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return tt.mockResp, nil
				},
			}

			c := &ClusterClient{
				hostURL:    "http://test.local",
				httpClient: mockClient,
			}

			err := c.CreateCluster(tt.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("CreateCluster() error = %v, should contain %v", err, tt.errContains)
			}
		})
	}
}
