/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"net/http"

	"github.com/segmentio/analytics-go"
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
