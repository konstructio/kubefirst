package addon

import (
	"log"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
)

func AddAddon(s string) {
	addons := viper.GetStringSlice("addons")
	if !slices.Contains(addons, s) {
		log.Printf("Adding addon on kubefirst file: %s", s)
		addons = append(addons, s)
		viper.Set("addons", addons)
	} else {
		log.Printf("Addon already on kubefirst file, nothing to do: %s", s)
	}
}

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
