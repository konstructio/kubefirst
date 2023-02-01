# Gitlab Platform Installation with the Kubefirst CLI

**Kubefirst** is the name of our command line tool that installs the Kubefirst platform and provides some platform 
conveniences.

It installs a fully automated platform of open source cloud native tools with a simple `init` and `create` command.

![](../../img/kubefirst/kubefirst-cluster-create.png)

### Prerequisites

- [To install kubefirst CLI](../overview.html#how-to-install-kubefirst-cli)

***AWS Prerequisites***

For your AWS cloud resources to provision we have just a couple prerequisites:    

1. An AWS account with billing enabled.

2. an established publicly routable hosted zone with dns ([docs](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/hosted-zones-working-with.html))

3. you'll need [AdministratorAccess](https://console.aws.amazon.com/iam/home?#/policies/arn:aws:iam::aws:policy/AdministratorAccessserviceLevelSummary) to your aws account ([docs](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys))


### Step 1 - `kubefirst init`

Then init your local setup providing values for the following flags:

| Flag              | Description                                                                                                                           | Example                   |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ------------------------- |
| --admin-email     | an email address that can be used for certificate renewal alerts and the gitlab root account                                          | yourname@yourcompany.com  |
| --cloud           | we only support aws, gcp coming soon                                                                                                  | aws                       |
| --hosted-zone-name| name of the platform's hosted zone domain - this will drive the URLs of your tools (gitlab.yourdomain.com, argocd.yourdomain.com, etc)| yourdomain.com            |
| --region          | name of the aws region in which to place your region specific resources                                                               | us-east-1                 |
| --profile         | name of the aws profile the cli should leverage                                                                                       | default                   |
| --cluster-name    | name of the cluster name, used to identify resources on cloud provider                                                                | your_cluster_name         |
| --s3-suffix       | unique identifier for s3 buckets                                                                                                      | you-s3-bucket-name        |
| --git-provider    | specify \"github\" or \"gitlab\" git provider                                                                                         | gitlab                    |
| --aws-nodes-spot  | nodes spot on AWS EKS compute nodes                                                                                                   | true                      |

```bash
kubefirst init \
--admin-email yourname@yourcompany.com \
--cloud aws \
--hosted-zone-name yourdomain.com \
--region us-east-1 \
--profile default \
--cluster-name your_cluster_name \
--s3-suffix you-s3-bucket-name \
--git-provider gitlab \
--aws-nodes-spot
```

#### Using a config file

```yaml
# config.yaml
config:
  admin-email: yourname@yourcompany.com
  hosted-zone-name: yourdomain.com
  region: us-east-1
  profile: default
  cluster-name: your_cluster_name
  s3-suffix: you-s3-bucket-name
  aws-nodes-spot: true
  cloud: aws  
  git-provider: gitlab  
```

```bash
kubefirst init -c config.yaml
```

The `init` process produces a directory of utilities, a state file, and some staged platform content that can now be 
found at `~/.kubefirst`. [Here](../../tooling/kubefirst-cli.md) you can find more details about `init` command.

### Step 2 - `kubefirst cluster create`

Now it's time to create the platform, to do so simply run

```
kubefirst cluster create
```

## What to do next

Continue your journey: 

- [Explore your installation](./explore.md)
- [Destroying](./destroy.md)
