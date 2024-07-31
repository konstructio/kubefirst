/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/kubefirst/internal/progress"
)

// displayFormattedClusterInfo uses tabwriter to pretty print information on clusters using
// the specified formatting
func displayFormattedClusterInfo(clusters []types.Cluster) {
	header := `
| NAME | CREATED AT | STATUS | TYPE | PROVIDER |
| --- | --- | --- | --- | --- |
	`
	content := ""
	for _, cluster := range clusters {
		content = content + fmt.Sprintf("|%s|%s|%s|%s|%s\n",
			cluster.ClusterName,
			cluster.CreationTimestamp,
			cluster.Status,
			cluster.ClusterType,
			cluster.CloudProvider,
		)
	}

	progress.Success(header + content)
}
