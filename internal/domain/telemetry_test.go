package domain

import (
	"github.com/denisbrodbeck/machineid"
	"reflect"
	"testing"
)

func TestNewTelemetry(t *testing.T) {

	machineId, err := machineid.ID()
	if err != nil {
		t.Error(err)
	}
	validTelemetry := Telemetry{MetricName: "test metric", Domain: "example.com", CLIVersion: "0.0.0", MachineId: machineId}

	type args struct {
		metricName string
		domain     string
		cliVersion string
		machineId  string
	}
	tests := []struct {
		name    string
		args    args
		want    Telemetry
		wantErr bool
	}{
		{
			name: "valid domain",
			args: args{
				metricName: "test metric",
				domain:     "https://example.com",
				cliVersion: "0.0.0",
				machineId:  machineId,
			},
			want:    validTelemetry,
			wantErr: false,
		},
		{
			name: "invalid domain",
			args: args{
				metricName: "test metric",
				domain:     "https://example-com",
				cliVersion: "0.0.0",
				machineId:  machineId,
			},
			want:    Telemetry{},
			wantErr: true,
		},
		{
			name: "empty domain (localhost)",
			args: args{
				metricName: "test metric",
				domain:     "",
				cliVersion: "0.0.0",
				machineId:  machineId,
			},
			want: Telemetry{
				MetricName: "test metric",
				Domain:     machineId,
				CLIVersion: "0.0.0",
				MachineId:  machineId,
			},
			wantErr: false,
		},
		{
			name: "missing telemetry name",
			args: args{
				metricName: "",
				domain:     "example.com",
				cliVersion: "0.0.0",
				machineId:  machineId,
			},
			want:    Telemetry{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTelemetry(tt.args.metricName, tt.args.domain, tt.args.cliVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTelemetry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTelemetry() got = %v, want %v", got, tt.want)
			}
		})
	}
}
