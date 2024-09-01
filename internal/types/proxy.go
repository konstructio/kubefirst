/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	apiTypes "github.com/konstructio/kubefirst-api/pkg/types"
)

type ProxyCreateClusterRequest struct {
	Body apiTypes.ClusterDefinition `bson:"body" json:"body"`
	Url  string                     `bson:"url" json:"url"`
}

type ProxyResetClusterRequest struct {
	Url string `bson:"url" json:"url"`
}
