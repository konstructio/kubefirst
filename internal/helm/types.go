/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package helm

type Repo struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type Release struct {
	AppVersion string `yaml:"app_version"`
	Chart      string `yaml:"chart"`
	Name       string `yaml:"name"`
	Namespace  string `yaml:"namespace"`
	Revision   string `yaml:"revision"`
	Status     string `yaml:"status"`
	Updated    string `yaml:"updated"`
}
