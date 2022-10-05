package handlers

import (
	"github.com/kubefirst/kubefirst/internal/services"
	"github.com/kubefirst/kubefirst/pkg"
	"reflect"
	"testing"
)

func TestNewTelemetry(t *testing.T) {

	// mocks
	httpMock := pkg.HTTPMock{}
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
				httpClient: httpMock,
				service:    mockedService,
			},
			want: TelemetryHandler{
				httpClient: httpMock,
				service:    mockedService,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTelemetry(tt.handler.httpClient, tt.handler.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTelemetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelemetryHandler_SendCountMetric(t *testing.T) {

	// mocks
	httpMock := pkg.HTTPMock{}
	segmentIOMock := pkg.SegmentIOMock{}

	mockedService := services.SegmentIoService{
		SegmentIOClient: segmentIOMock,
	}

	type args struct {
		metricName string
		domain     string
		cliVersion string
	}
	tests := []struct {
		name    string
		handler TelemetryHandler
		args    args
		wantErr bool
	}{{
		name: "should pass, its all correct",
		handler: TelemetryHandler{
			httpClient: httpMock,
			service:    mockedService,
		},
		args: args{
			metricName: "test-metric-name",
			domain:     "example.com",
			cliVersion: "0.0.1-test",
		},
		wantErr: false,
	},
		{
			name: "should fail when metric name is empty",
			handler: TelemetryHandler{
				httpClient: httpMock,
				service:    mockedService,
			},
			args: args{
				metricName: "",
				domain:     "example.com",
				cliVersion: "0.0.1-test",
			},
			wantErr: true,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := TelemetryHandler{
				httpClient: tt.handler.httpClient,
				service:    tt.handler.service,
			}
			if err := handler.SendCountMetric(tt.args.metricName, tt.args.domain, tt.args.cliVersion); (err != nil) != tt.wantErr {
				t.Errorf("EnqueueCountMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
