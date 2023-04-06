/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package segment

import (
	"github.com/kubefirst/kubefirst/pkg"
	"github.com/segmentio/analytics-go"
)

var Client SegmentClient = SegmentClient{
	Client: newSegmentClient(),
}

func newSegmentClient() analytics.Client {

	client := analytics.New(pkg.SegmentIOWriteKey)

	return client
}
