# Kubefirst Command Line

Kubefirst provides a CLI to empower the whole previsioning cluster management process. The basic usage of the CLI is to
fully create the Kubefirst previsioning process, but there are also many other handful options to backup, validation and
cluster destroy.

## Kubefirst CLI available commands

- [clean](#clean)
- [init](#init)
- [cluster](#cluster-create)
- [destroy](#destroy)
- [backupSSL](#backupssl)
- [restoreSSL](#restoressl)
- [state](#state)

### clean

Kubefirst creates files, folders and cloud buckets during installation at your environment. This command removes and
re-create Kubefirst base files. To destroy cloud resources you need to specify additional flags (--destroy-buckets)

```
Usage:
kubefirst clean [flags]

Flags:
--destroy-buckets   destroy buckets created by init cmd
--destroy-confirm   confirm destroy operation (to be used during automation process)
-h, --help              help for clean
--rm-logs           remove logs folder
```

### init

Init command will prepare the installation, and create the initial resources that will support the installation when 
`kubefirst cluster create` is be called.

```
Usage:
kubefirst init [flags]

Flags:
--admin-email string        the email address for the administrator as well as for lets-encrypt certificate emails
--aws-assume-role string    instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html
--aws-nodes-spot            nodes spot on AWS EKS compute nodes
--clean                     delete any local kubefirst content ~/.kubefirst, ~/.k1
--cloud string              the cloud to provision infrastructure in
--cluster-name string       the cluster name, used to identify resources on cloud provider (default "kubefirst")
--dry-run                   set to dry-run mode, no changes done on cloud provider selected
--github-host string        Github URL (default "github.com")
--github-org string         Github Org of repos
--github-owner string       Github owner of repos
--github-user string        Github user
--gitops-branch string      version/branch used on git clone - former: version-gitops flag
--gitops-owner string       git owner of gitops, this may be a user or a org to support forks for testing (default "kubefirst")
--gitops-repo string        version/branch used on git clone (default "gitops")
-h, --help                      help for init
--hosted-zone-name string   the domain to provision the kubefirst platform in
--metaphor-branch string    version/branch used on git clone - former: version-gitops flag
--profile string            AWS profile located at ~/.aws/config (default "default")
--region string             the region to provision the cloud resources in (default "eu-west-1")
--s3-suffix string          unique identifier for s3 buckets
--template-tag string       fallback tag used on git clone.
Details: if "gitops-branch" is provided, branch("gitops-branch") has precedence and installer will attempt to clone branch("gitops-branch") first,
if it fails, then fallback it will attempt to clone the tag provided at "template-tag" flag (default "1.8.6")
--use-telemetry             installer will not send telemetry about this installation (default true)
```

### cluster create

Cluster Level operations like create a new cluster provisioned with all kubefirst goodies.

```
Usage:
kubefirst cluster [flags]
kubefirst cluster [command]

Available Commands:
create        create a kubefirst management cluster
create-github create a kubefirst management cluster with github as Git Repo
create-gitlab create a kubefirst management cluster

Flags:
-h, --help   help for cluster

Use "kubefirst cluster [command] --help" for more information about a command.
```

### destroy

Destroy the kubefirst management cluster and all of the components in kubernetes.

Optional: skip gitlab terraform if the registry has already been deleted.

```
Usage:
kubefirst destroy [flags]

Flags:
--dry-run                 set to dry-run mode, no changes done on cloud provider selected
-h, --help                    help for destroy
--skip-base-terraform     whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist
--skip-delete-register    whether to skip deletion of register application
--skip-gitlab-terraform   whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist
```

### backupSSL
This command create a backup of secrets from cert manager certificates to bucket named kubefirst-<DOMAIN> where can be 
used on provisioning phase with the flag --recycle-ssl

```
Usage:
kubefirst backupSSL [flags]

Flags:
-h, --help   help for backupSSL
```

### restoreSSL

Restore the backup SSL that is stored in the S3 bucket and avoid re-issuing new certificates.

```
Usage:
kubefirst restoreSSL [flags]

Flags:
--dry-run         Set to dry-run mode, no changes done on cloud provider selected.
-h, --help            help for restoreSSL
--use-telemetry   installer will not send telemetry about this installation (default true)
```

### state

Kubefirst configuration can be handed over to another user by pushing the Kubefirst config files to a S3 bucket.

```
Usage:
kubefirst state [flags]

Flags:
--bucket-name string   set the bucket name to store the Kubefirst config file
-h, --help                 help for state
--pull                 pull Kubefirst config file to the S3 bucket
--push                 push Kubefirst config file to the S3 bucket
--region string        Set S3 bucket region.
```

