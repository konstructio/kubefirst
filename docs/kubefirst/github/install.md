# Platform Installation with the Kubefirst CLI

`kubefirst` is the name of our command line tool that installs the kubefirst platform and provides some platform conveniences.

It installs a fully automated platform of open source cloud native tools with a simple `init` and `create` command.

![](../../img/todo.jpeg)

`todo: new image needed for github install. DESIGN-251`

![](../../img/kubefirst/kubefirst-cluster-create.png)

### Prerequesites

**AWS Prerequesites**

1. Create an AWS account with billing enabled

2. Establish a public hosted zone with dns routing established ([docs](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html))

3. Connect with [AdministratorAccess](https://console.aws.amazon.com/iam/home?#/policies/arn:aws:iam::aws:policy/AdministratorAccessserviceLevelSummary) IAM credentials to your aws account ([docs](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys))

**GitHub Prerequesites**

1. Establish a [GitHub organization](https://docs.github.com/en/organizations/collaborating-with-groups-in-organizations/creating-a-new-organization-from-scratch)

### Step 1 - Download

Download the latest kubefirst cli.

**installation using homebrew**

```
brew install kubefirst/tools/kubefirst
```

**installation using other methods**

There are a number of other ways to install kubefirst for different operating systems, architectures, and containerized environments. Please see our [installation readme](https://github.com/kubefirst/kubefirst/blob/main/build/README.md) for details.

### Step 2 - `kubefirst init`

Then init your local setup providing values for the following flags:

| Flag | Description | Example |
| ---- | ----------- | ------- |
| --admin-email | an email address that can be used for certificate renewal alerts and the gitlab root account | yourname@yourcompany.com |
| --cloud | we only support aws, gcp coming soon | aws |
| --hosted-zone-name | name of the platform's hosted zone domain - this will drive the URLs of your tools (gitlab.yourdomain.com, argocd.yourdomain.com, etc) | yourdomain.com |
| --region | name of the aws region in which to place your region specific resources | us-east-1 |
| --profile | name of the aws profile the cli should leverage | default |

```
kubefirst init --admin-email yourname@yourcompany.com --cloud aws --hosted-zone-name yourdomain.com --region us-east-1 --profile default
```

The `init` process produces a directory of utilities, a state file, and some staged platform content that can now be found at `~/.kubefirst`
<!-- TODO: check final state file name above - state file collides with directory -->

### Step 3 - `kubefirst cluster create`

Now it's time to create the platform, to do so simply run

```
kubefirst cluster create
```
<!-- TODO: check final state command above - talk through stack vs cluster with team -->
