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
)

// displayFormattedClusterInfo uses tabwriter to pretty print information on clusters using
// the specified formatting
func displayFormattedClusterInfo(clusters []types.Cluster) error {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 1, ' ', tabwriter.Debug)

	fmt.Fprint(tw, "NAME\tCREATED AT\tSTATUS\tTYPE\tPROVIDER\n")
	for _, cluster := range clusters {
		_, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			cluster.ClusterName,
			cluster.CreationTimestamp,
			cluster.Status,
			cluster.ClusterType,
			cluster.CloudProvider,
		)

		if err != nil {
			return fmt.Errorf("failed to write to tabwriter: %w", err)
		}
	}

	// TODO: Handle for non-bubbletea
	// progress.Success(buf.String())
	return nil
}
