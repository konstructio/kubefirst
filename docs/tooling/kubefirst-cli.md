# Kubefirst Command Line

Kubefirst provides a CLI to empower the whole previsioning cluster management process. The basic usage of the CLI is to
fully create the Kubefirst provisioning process, but there are also many other helpful options to backup, validate and
destroy.

## Kubefirst CLI Available Commands

- [clean](#clean)
- [init](#init)
- [cluster](#cluster-create)
- [destroy](#destroy)
- [backupSSL](#backupssl)
- [restoreSSL](#restoressl)
- [state](#state)

### clean

Kubefirst creates files, folders and cloud buckets during installation at your environment. This command removes and
re-creates Kubefirst base files. To destroy cloud resources you need to specify additional flags (--destroy-buckets)

```
Usage:
kubefirst clean [flags]

Flags:
--destroy-buckets   destroy buckets created by init cmd
--destroy-confirm   confirm destroy operation (to be used during automation process)
-h, --help          help for clean
--preserve-tools    preserve all downloaded tools (avoid re-downloading)
--rm-logs           remove logs folder
```

### init

The **init** command will prepare the installation and create the initial resources that will support the installation when 
`kubefirst cluster create` is called.

```
Usage:
kubefirst init [flags]

Flags:
--admin-email string        The email address for the administrator as well as for Lets-Encrypt certificate emails.
--aws-assume-role string    Instead of using AWS IAM user credentials, AWS AssumeRole feature generates role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html.
--aws-nodes-graviton        nodes Graviton on AWS EKS compute nodes, more info [https://aws.amazon.com/ec2/graviton/]
--aws-nodes-spot            Nodes spot on AWS EKS compute nodes.
--clean                     Delete any local Kubefirst content ~/.kubefirst, ~/.k1.
--cloud string              The cloud in which to provision infrastructure.
--cluster-name string       The cluster name, used to identify resources on cloud provider (default "kubefirst").
--dry-run                   Set to dry-run mode - no changes done on cloud provider selected.
--github-host string        Github URL (default "github.com")
--github-org string         Github Org of repos
--github-owner string       Github owner of repos
--github-user string        Github user
--gitops-branch string      Version/branch used on Git clone - former: version-gitops flag.
--gitops-owner string       Git owner of GitOps - this may be a user or an org to support forks for testing (default "kubefirst").
--gitops-repo string        Version/branch used on Git clone (default "gitops").
-h, --help                  Help for init.
--hosted-zone-name string   The domain in which to provision the Kubefirst platform.
--metaphor-branch string    Version/branch used on Git clone - former: version-gitops flag.
--profile string            AWS profile located at ~/.aws/config (default "default").
--region string             The region in which to provision the cloud resources (default "eu-west-1").
--s3-suffix string          Unique identifier for S3 buckets.
--template-tag string       Fallback tag used on Git clone.
Details: If "gitops-branch" is provided, branch("gitops-branch") has precedence and the installer will attempt to clone the branch("gitops-branch") first,
if it fails, then it will attempt to clone the tag provided at "template-tag" flag (default "1.8.6").
--use-telemetry             Installer won't send telemetry data if --use-telemetry=false is set (default true).
```

### cluster create

Cluster Level operations like **create a new cluster** provisioned with all Kubefirst goodies.

```
Usage:
kubefirst cluster [flags]
kubefirst cluster [command]

Available Commands:
create        Create a Kubefirst management cluster.
create-github Create a Kubefirst management cluster with GitHub as Git Repo.
create-gitlab Create a Kubefirst management cluster.

Flags:
-h, --help   Help for cluster.

Use "kubefirst cluster [command] --help" for more information about a command.
```

### destroy

Destroy the Kubefirst management cluster and all of the components in Kubernetes.

Optional: Skip GitLab Terraform if the registry has already been deleted.

```
Usage:
kubefirst cluster destroy [flags]

Flags:
--dry-run                 Set to dry-run mode, no changes done on cloud provider selected.
-h, --help                Help for destroy.
--skip-base-terraform     Whether to skip the Terraform destroy against the base install. Note: If you already deleted the registry, it doesn't exist.
--skip-delete-register    Whether to skip the deletion of register application.
--skip-gitlab-terraform   Whether to skip the Terraform destroy against GitLab. Note: If you already deleted the registry, it doesn't exist.
```

### backupSSL
This command creates a backup of secrets from cert manager certificates to a bucket named Kubefirst-<DOMAIN> where secrets can be 
used on the provisioning phase with the flag --recycle-ssl.

```
Usage:
kubefirst backupSSL [flags]

Flags:
-h, --help   Help for backupSSL.
```

### restoreSSL

Restore the backup SSL that is stored in the S3 bucket and avoid re-issuing new certificates.

```
Usage:
kubefirst restoreSSL [flags]

Flags:
--dry-run         Set to dry-run mode, no changes done on cloud provider selected.
-h, --help        Help for restoreSSL.
--use-telemetry   Installer won't send telemetry data if --use-telemetry=false is set (default true).
```

### state

Kubefirst configuration can be handed over to another user by pushing the Kubefirst config files to a S3 bucket.

```
Usage:
kubefirst state [flags]

Flags:
--bucket-name string   Set the bucket name to store the Kubefirst config file.
-h, --help             Help for state.
--pull                 Pull the Kubefirst config file from the S3 bucket.
--push                 Push Kubefirst config file to the S3 bucket.
--region string        Set S3 bucket region.
```

