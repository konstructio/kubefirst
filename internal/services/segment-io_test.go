package services

import (
	"github.com/kubefirst/kubefirst/pkg"
	"reflect"
	"testing"
)

func TestNewSegmentIoService(t *testing.T) {

	segmentIOMock := pkg.SegmentIOMock{}

	tests := []struct {
		name    string
		service SegmentIoService
		want    SegmentIoService
	}{{
		name: "new SegmentIO service with valid segmentIO client",
		service: SegmentIoService{
			SegmentIOClient: segmentIOMock,
		},
		want: SegmentIoService{
			SegmentIOClient: segmentIOMock,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSegmentIoService(tt.service.SegmentIOClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSegmentIoService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSegmentIoService_SendCountMetric(t *testing.T) {

	segmentIOMock := pkg.SegmentIOMock{}

	type args struct {
		metricName string
		domain     string
		cliVersion string
	}
	tests := []struct {
		name    string
		service SegmentIoService
		args    args
		wantErr bool
	}{
		{
			name: "metric sent with success",
			service: SegmentIoService{
				SegmentIOClient: segmentIOMock,
			},
			args: args{
				metricName: "metric-name-test",
				domain:     "example.com",
				cliVersion: "0.0.1-test",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := SegmentIoService{
				SegmentIOClient: tt.service.SegmentIOClient,
			}
			if err := service.EnqueueCountMetric(tt.args.metricName, tt.args.domain, tt.args.cliVersion); (err != nil) != tt.wantErr {
				t.Errorf("EnqueueCountMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
