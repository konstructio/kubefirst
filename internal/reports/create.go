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
	ArgoCDUrl         string
	ArgoCDUsername    string
	ArgoCDPassword    string
	VaultUrl          string
	VaultToken        string
}

func BuildCreateHandOffReport(clusterData CreateHandOff) bytes.Buffer {

	var handOffData bytes.Buffer
	handOffData.WriteString(strings.Repeat("-", 70))
	handOffData.WriteString(fmt.Sprintf("\nCluster %q is up and running!:\n", clusterData.ClusterName))
	handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n\n--- AWS ")
	handOffData.WriteString(strings.Repeat("-", 62))
	handOffData.WriteString(fmt.Sprintf("\n AWS Account Id: %s\n", clusterData.AwsAccountId))
	handOffData.WriteString(fmt.Sprintf(" AWS hosted zone name: %s\n", clusterData.AwsHostedZoneName))
	handOffData.WriteString(fmt.Sprintf(" AWS region: %s\n", clusterData.AwsRegion))
	handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n\n--- ArgoCD ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s\n", clusterData.ArgoCDUrl))
	handOffData.WriteString(fmt.Sprintf(" username: %s\n", clusterData.ArgoCDUsername))
	handOffData.WriteString(fmt.Sprintf(" password: %s\n", clusterData.ArgoCDPassword))
	handOffData.WriteString(strings.Repeat("-", 70))

	handOffData.WriteString("\n\n--- Vault ")
	handOffData.WriteString(strings.Repeat("-", 59))
	handOffData.WriteString(fmt.Sprintf("\n URL: %s\n", clusterData.VaultUrl))
	handOffData.WriteString(fmt.Sprintf(" token: %s\n", clusterData.VaultToken))
	handOffData.WriteString(strings.Repeat("-", 70))

	return handOffData

}
