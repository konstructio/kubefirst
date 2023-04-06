/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	"bytes"
	"fmt"
	"strings"
)

type CreateHandOff struct {
	AwsAccountId      string
	AwsHostedZoneName string
	AwsRegion         string
	ClusterName       string

	GitlabURL      string
	GitlabUser     string
	GitlabPassword string

	RepoGitops   string
	RepoMetaphor string

	VaultURL   string
	VaultToken string

	ArgoCDURL      string
	ArgoCDUsername string
	ArgoCDPassword string

	ArgoWorkflowsURL string

	AtlantisURL string

	ChartMuseumURL string

	MetaphorDevURL        string
	MetaphorStageURL      string
	MetaphorProductionURL string
}

func BuildCreateHandOffReport(clusterData CreateHandOff) bytes.Buffer {

	var handOffData bytes.Buffer
	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString(fmt.Sprintf("\nCluster %q is up and running!:", clusterData.ClusterName))
	handOffData.WriteString("\nThis information is available at $HOME/.kubefirst ")
	handOffData.WriteString("\nPress ESC to leave this screen and return to shell.")
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- AWS ")
	handOffData.WriteString(strings.Repeat("-", 62))
	handOffData.WriteString(fmt.Sprintf("\n AWS Account Id: %s", clusterData.AwsAccountId))
	handOffData.WriteString(fmt.Sprintf("\n AWS hosted zone name: %s", clusterData.AwsHostedZoneName))
	handOffData.WriteString(fmt.Sprintf("\n AWS region: %s", clusterData.AwsRegion))
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- GitLab ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n username: %s", clusterData.GitlabUser))
	handOffData.WriteString(fmt.Sprintf("\n password: %s", clusterData.GitlabPassword))
	handOffData.WriteString("\n Repos: ")
	handOffData.WriteString(fmt.Sprintf("\n  %s", clusterData.RepoGitops))
	handOffData.WriteString(fmt.Sprintf("\n  %s", clusterData.RepoMetaphor))
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 60))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", clusterData.VaultURL))
	handOffData.WriteString(fmt.Sprintf("\n token: %s", clusterData.VaultToken))
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", clusterData.ArgoCDURL))
	handOffData.WriteString(fmt.Sprintf("\n username: %s", clusterData.ArgoCDUsername))
	handOffData.WriteString(fmt.Sprintf("\n password: %s", clusterData.ArgoCDPassword))
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- Argo Workflows ")
	handOffData.WriteString(strings.Repeat("-", 51))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", clusterData.ArgoWorkflowsURL))
	handOffData.WriteString("\n sso credentials only ")
	handOffData.WriteString("\n * sso enabled ")
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- Atlantis ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s", clusterData.AtlantisURL))
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- Museum ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s\n", clusterData.ChartMuseumURL))
	handOffData.WriteString(" see vault for credentials ")
	//handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n--- Metaphor ")
	handOffData.WriteString(strings.Repeat("-", 57))
	handOffData.WriteString(fmt.Sprintf("\n Development: %s", clusterData.MetaphorDevURL))
	handOffData.WriteString(fmt.Sprintf("\n Staging: %s", clusterData.MetaphorStageURL))
	handOffData.WriteString(fmt.Sprintf("\n Production:  %s\n", clusterData.MetaphorProductionURL))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData

}

func BuildSection() bytes.Buffer {
	var handOffData bytes.Buffer
	return handOffData
}
