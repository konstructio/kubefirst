### KubeFirst Allows You to Opt out of Data Collection

Kubefirst collects data around users in order to optimize future releases. By collecting metrics on what type of clusters are being deployed and how they are being used, Kubefirst prioritizes the features that are being used across the majority of the user base. While we rely on this data to make improvements to the platform, you are always allowed to opt out for any reason.

## What Metrics are collected?

- CLIVersion:      The version of CLI being used
- ClusterType:     The type of cluster being created (local, AWS, etc.)
- ClusterId:       The ID of the cluster being created
- Domain:          The domain of the cluster being created
- KubeFirstTeam:   Whether or not you are a Kubefirst teammate

## How to Opt Out

When installing your KubeFirst cluster through the cli, append the `--use-telemetry=false` flag to opt ourself out of this process.

