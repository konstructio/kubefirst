/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package launch

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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

// setupLaunchConfigFile
func setupLaunchConfigFile(dir string) error {
	viperConfigFile := fmt.Sprintf("%s/.launch", dir)

	if _, err := os.Stat(viperConfigFile); errors.Is(err, os.ErrNotExist) {
		log.Debugf("launch config file not found, creating a blank one: %s", viperConfigFile)
		err = os.WriteFile(viperConfigFile, []byte(""), 0700)
		if err != nil {
			return fmt.Errorf("unable to create blank config file, error is: %s", err)
		}
	}

	viper.SetConfigFile(viperConfigFile)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv() // read in environment variables that match

	// if a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("unable to read config file, error is: %s", err)
	}

	return nil
}
