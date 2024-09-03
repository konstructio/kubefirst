/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/konstructio/kubefirst/internal/progress"
)

// displayFormattedClusterInfo uses tabwriter to pretty print information on clusters using
// the specified formatting
func displayFormattedClusterInfo(clusters []types.Cluster) {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprint(tw, "NAME\tCREATED AT\tSTATUS\tTYPE\tPROVIDER\n")
	for _, cluster := range clusters {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			cluster.ClusterName,
			cluster.CreationTimestamp,
			cluster.Status,
			cluster.ClusterType,
			cluster.CloudProvider,
		)
	}

	progress.Success(buf.String())
}
