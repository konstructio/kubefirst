/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package segment

import (
	"github.com/segmentio/analytics-go"
)

type SegmentClient struct {
	Client analytics.Client
}
