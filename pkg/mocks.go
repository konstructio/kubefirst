package pkg

import (
	"github.com/segmentio/analytics-go"
	"net/http"
)

type HTTPMock struct{}

func (httpMock HTTPMock) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

type SegmentIOMock struct{}

func (segmentIOMock SegmentIOMock) Close() error {
	return nil
}

func (segmentIOMock SegmentIOMock) Enqueue(message analytics.Message) error {
	return nil
}
