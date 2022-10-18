package handlers

import (
	"github.com/kubefirst/kubefirst/internal/domain"
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"reflect"
	"testing"
)

func TestNewTelemetry(t *testing.T) {

	// mocks
	segmentIOMock := pkg.SegmentIOMock{}

	mockedService := services.SegmentIoService{
		SegmentIOClient: segmentIOMock,
	}

	tests := []struct {
		name    string
		handler TelemetryHandler
		want    TelemetryHandler
	}{
		{
			name: "newTelemetry",
			handler: TelemetryHandler{
				service: mockedService,
			},
			want: TelemetryHandler{
				service: mockedService,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTelemetryHandler(tt.handler.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTelemetryHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelemetryHandler_SendCountMetric(t *testing.T) {

	validTelemetry := domain.Telemetry{MetricName: "test metric", Domain: "example.com", CLIVersion: "0.0.0"}

	// mocks
	segmentIOMock := pkg.SegmentIOMock{}

	mockedService := services.SegmentIoService{
		SegmentIOClient: segmentIOMock,
	}

	type fields struct {
		service services.SegmentIoService
	}
	type args struct {
		telemetry domain.Telemetry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "valid telemetry",
			fields:  fields{service: mockedService},
			args:    args{telemetry: validTelemetry},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := TelemetryHandler{
				service: tt.fields.service,
			}
			if err := handler.SendCountMetric(tt.args.telemetry); (err != nil) != tt.wantErr {
				t.Errorf("SendCountMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
