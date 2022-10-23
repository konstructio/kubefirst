package domain

import (
	"reflect"
	"testing"
)

func TestNewTelemetry(t *testing.T) {

	validTelemetry := Telemetry{MetricName: "test metric", Domain: "example.com", CLIVersion: "0.0.0"}

	type args struct {
		metricName string
		domain     string
		CLIVersion string
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
				CLIVersion: "0.0.0",
			},
			want:    validTelemetry,
			wantErr: false,
		},
		{
			name: "invalid domain",
			args: args{
				metricName: "test metric",
				domain:     "https://example-com",
				CLIVersion: "0.0.0",
			},
			want:    Telemetry{},
			wantErr: true,
		},
		{
			name: "empty domain (localhost)",
			args: args{
				metricName: "test metric",
				domain:     "",
				CLIVersion: "0.0.0",
			},
			want:    Telemetry{},
			wantErr: false,
		},
		{
			name: "missing telemetry name",
			args: args{
				metricName: "",
				domain:     "example.com",
				CLIVersion: "0.0.0",
			},
			want:    Telemetry{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTelemetry(tt.args.metricName, tt.args.domain, tt.args.CLIVersion)
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
