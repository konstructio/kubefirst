package addon

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/viper"
)

func ListAddons() {
	addonsInstalled := viper.GetStringSlice("addons")
	addonsSupported := getSupported()
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Addon Name", "Installed?"})
	for _, addon := range addonsInstalled {
		t.AppendRows([]table.Row{
			{addon, "true"},
		})
	}
	t.AppendSeparator()
	t.AppendRow([]interface{}{"Addons Available", "Supported by"})
	t.AppendSeparator()
	for _, addon := range addonsSupported {
		t.AppendRows([]table.Row{
			{addon, "kubeshop/kubefirst"},
		})
	}

	t.Render()
}

func getSupported() []string {
	var addonsSupported []string
	addonsSupported = append(addonsSupported, "kusk")
	return addonsSupported
}

func EnableAddon(name string) error {
	// TODO
	return nil
}

func DisableAddon(name string) error {
	// TODO
	return nil
}
