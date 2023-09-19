/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"fmt"
	"os"
	"text/tabwriter"
)

// displayFormattedClusterInfo uses tabwriter to pretty print information on clusters using
// the specified formatting
func displayFormattedClusterInfo(clusters []map[string]interface{}) error {
	// A friendly warning before we proceed
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "NAME\tCREATED AT\tSTATUS\tTYPE\tPROVIDER")
	for _, cluster := range clusters {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			cluster["cluster_name"],
			cluster["creation_timestamp"],
			cluster["status"],
			cluster["cluster_type"],
			cluster["cloud_provider"],
		)
	}
	err := writer.Flush()
	if err != nil {
		return fmt.Errorf("error closing buffer: %s", err)
	}

	return nil
}
