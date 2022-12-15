# Destroying your Kubefirst local platform

## Automated Teardown

Before you attempt to recreate a Kubefirst local platform, you'll need to destroy your k3d cluster and the git repositories that we created for you using this command:

```bash
kubefirst local destroy
```

## Manual Teardown

If the above command fails to complete due to unforeseen circumstances, you can then manually delete the git repositories named:

- gitops
- metaphor-frontend (only exists if you complete Kubefirst local provisioning)

You can then manually delete the k3d cluster with the command `k3d cluster delete kubefirst` or `~/.k1/tools/k3d cluster delete kubefirst` if you don't have k3d installed.

## Localhost file cleanup

You can clean kubefirst files from your localhost by running

```bash
kubefirst clean
```

This command will remove the following content:

- `~/.kubefirst`
- `~/.k1/*`
