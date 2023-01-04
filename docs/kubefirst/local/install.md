# Local Installation with the Kubefirst CLI

**Kubefirst** is the name of our command line tool that installs the Kubefirst platform to your local or cloud environment.

To use the local version of Kubefirst, you will need to have [Docker installed](https://docs.docker.com/get-docker/). You will also need a GitHub account: GitLab for local, and local git repositories are not supported yet.

![Kubefirst local installation diagram](../../img/kubefirst/local/kubefirst-cluster-create.png)

## Step 1 - Download (or upgrade) the Kubefirst CLI

```shell
brew install kubefirst/tools/kubefirst
```

There are a few other ways to install Kubefirst for different operating systems, architectures, and containerized environments. See our [installation README](https://github.com/kubefirst/kubefirst/blob/main/build/README.md) for non-brew details.

To upgrade an existing Kubefirst install to the latest version run

```shell
brew update
brew upgrade kubefirst
```

## Step 2 - Create your new local cluster

To create a new Kubefirst cluster locally, run

```shell
kubefirst local
```

### GitHub Authorization during install

The `kubefirst local` command will request that you authorize the Kubefirst CLI to be able to manage Git repositories in your personal GitHub account. It needs this access to add 4 repositories to your account. To do this you'll copy a code that's sent to your terminal, paste it in a GitHub auth page that opens, and hit the `Authorize` button.

#### Why the Kubefirst cli needs this access

The [gitops repo](https://github.com/kubefirst/gitops-template) that we create for you will power your local Kubefirst platform. The [metaphor](https://github.com/kubefirst/metaphor-template), [metaphor-go](https://github.com/kubefirst/metaphor-go-template), and [metaphor-frontend](https://github.com/kubefirst/metaphor-frontend-template) repos are your microservices examples, which demonstrate how to publish and gitops-deliver applications to your new development, staging, and production namespaces in your new local cluster.

## After installation

After the ~5 minutes installation, your browser will launch a new tab to the [Kubefirst Console application](https://github.com/kubefirst/console), which will help you navigate your new suite of tools running in your local k3d cluster.
