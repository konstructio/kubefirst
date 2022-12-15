# Automated Teardown (Plan A)

Before you attempt to recreate a kubefirst local platform you'll need to destroy your k3d cluster and the git repositories that we create for you. Under normal circumstances, you can delete these with the command:

```bash
kubefirst local destroy
```

# Manual Teardown (Plan B)

If the above command fails to complete due to unforseen circumstances in your execution, you can then manually delete the git repositories named
- gitops
- metaphor (only exists if you complete kubefirst local provisioning)

You can then manually delete the k3d cluster with the command `k3d cluster delete kubefirst`.

# Localhost file cleanup

You can clean kubefirst files from your localhost by running

```bash
kubefirst clean
```

This autoamted will remove the following content:
- `~/.kubefirst`
- `~/.k1/*`
