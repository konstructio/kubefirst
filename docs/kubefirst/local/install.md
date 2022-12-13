# Local Installation with the Kubefirst CLI

**Kubefirst** is the name of our command line tool that installs the Kubefirst platform to your local or cloud environment.

![](../../img/kubefirst/local/kubefirst-cluster-create.png)

### Step 1 - Download (or upgrade) the kubefirst CLI

```
brew install kubefirst/tools/kubefirst
```

There are a few other ways to install Kubefirst for different operating systems, architectures, and containerized environments. See our [installation readme](https://github.com/kubefirst/kubefirst/blob/main/build/README.md) for non-brew details.

To upgrade an existing kubefirst install to the latest version run
```
brew upgrade kubefirst
```

### Step 2 - Create your new local cluster

To create a new kubefirst locally run
```
kubefirst local
```

### GitHub Authorization during install

The `kubefirst local` command will request that you authorize the kubefirst CLI to be able to manage Git repositories in your personal GitHub account. It needs this access to add 4 repositories to your personal github account. To do this you'll copy a code that's sent to your terminal, paste it in a GitHub auth page that opens, and hit the Authorize button.

### Why the kubefirst cli needs this access

The `gitops` repo that we create for you will power the local kubefirst platform. The `metaphor`, `metaphor-go`, and `metaphor-frontend` repos are your example sample microservices which will demonstrate how to publish and gitops-deliver applications to your new development, staging, and production namespaces in your new local cluster.

### After installation

After the ~5 minute installation, your browser launch a new tab to the kubefirst console application, which will help you navigate to your new suite of tools running in your local k3d cluster.
