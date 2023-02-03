# Civo Platform Installation with the Kubefirst CLI

**Kubefirst** is the name of our command line tool that installs the Kubefirst platform to your local or cloud environment.

It installs a fully automated platform of open source cloud native tools with a simple `kubefirst civo create` command.

![](../../img/kubefirst/civo/kubefirst-cluster-create.png)

### CIVO Prerequisites

For your CIVO cloud resources to provision we have a couple prerequisites:    

1. A CIVO account with billing enabled. # note when you get an account you get $200 in credits!

2. an established publicly accessiblec dns ([docs](https://docs.CIVO.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html))

3. you'll need [AdministratorAccess](https://console.CIVO.amazon.com/iam/home?#/policies/arn:CIVO:iam::CIVO:policy/AdministratorAccessserviceLevelSummary) to your aws account ([docs](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys))


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
kubefirst civo create
```

The `kubefirst` cli will produce a directory of utilities, a state file, and some staged platform content that can now be 
found at `~/.kubefirst` and `~/.k1`.
<!-- TODO: check final state file name above - state file collides with directory -->
