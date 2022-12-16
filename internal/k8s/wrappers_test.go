package k8s

import (
	"github.com/kubefirst/kubefirst/configs"
	"testing"
)

func TestCreateSecretsFromCertificatesForLocalWrapper(t *testing.T) {

	config := configs.ReadConfig()

	type args struct {
		config     *configs.Config
		disableTLS bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "skip if --disable-tls is true",
			args: args{
				config:     config,
				disableTLS: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateSecretsFromCertificatesForLocalWrapper(tt.args.config, tt.args.disableTLS); (err != nil) != tt.wantErr {
				t.Errorf("CreateSecretsFromCertificatesForLocalWrapper() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
