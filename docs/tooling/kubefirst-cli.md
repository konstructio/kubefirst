# Kubefirst Command Line

Kubefirst provides a CLI to empower the whole previsioning cluster management process. The basic usage of the CLI is to
fully create the Kubefirst provisioning process, but there are also many other helpful options to backup, validate and
destroy.

## Kubefirst CLI Available Commands


### Local

- [local](#local)
- [local destroy](#local-destroy)

### Cloud

- [init](#init)
- [cluster](#cluster-create)
- [destroy](#cluster-destroy)
- [backupSSL](#backupssl)
- [restoreSSL](#restoressl)

### Tooling 

- [clean](#clean)
- [state](#state)



### base

`kubefirst`:

```bash 
kubefirst management cluster installer provisions an
  open source application delivery platform in under an hour.
  checkout the docs at docs.kubefirst.io.

Usage:
  kubefirst [flags]
  kubefirst [command]

Available Commands:
  action       A brief description of your command
  addon        Addon Command - to manage Kubefirst supported addons
  argocdSync   request ArgoCD to synchronize applications
  backupSSL    Backup Secrets (cert-manager/certificates) to bucket kubefirst-<DOMAIN>
  checktools   use to check compatibility of .kubefirst/tools
  clean        removes all kubefirst resources created with the init command
  cluster      Used to manage cluster level operations
  completion   Generate the autocompletion script for the specified shell
  help         Help about any command
  info         provides general Kubefirst setup data
  init         Initialize your local machine to execute `create`
  local        Kubefirst localhost installation
  post-install starts post install process
  restoreSSL   Restore SSL certificates from a previous install
  state        push and pull Kubefirst configuration to S3 bucket
  version      print the version number for kubefirst-cli

Flags:
  -h, --help   help for kubefirst

Use "kubefirst [command] --help" for more information about a command.
```

### clean

Kubefirst creates files, folders and cloud buckets during installation at your environment. This command removes and
re-creates Kubefirst base files. To destroy cloud resources you need to specify additional flags (--destroy-buckets)

```bash
Kubefirst creates files, folders and cloud buckets during installation at your environment. This command removes and
re-create Kubefirst base files. To destroy cloud resources you need to specify additional flags (--destroy-buckets)

Usage:
  kubefirst clean [flags]

Flags:
      --destroy-buckets   destroy buckets created by init cmd
      --destroy-confirm   when detroy-buckets flag is provided, we must provide this flag as well to confirm the destroy operation
  -h, --help              help for clean
      --rm-logs           remove logs folder
```

### init

The **init** command will prepare the installation and create the initial resources that will support the installation when 
`kubefirst cluster create` is called.

```bash
Initialize the required resources to provision a full Cloud environment. At this step initial resources are
validated and configured.

Usage:
  kubefirst init [flags]

Flags:
      --addons strings            the name of addon to enable on create cluster:
                                    --addon foo or --addon foo,bar for example
      --admin-email string        the email address for the administrator as well as for lets-encrypt certificate emails
      --aws-assume-role string    instead of using AWS IAM user credentials, AWS AssumeRole feature generate role based credentials, more at https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html
      --aws-nodes-graviton        nodes Graviton on AWS EKS compute nodes, more info [https://aws.amazon.com/ec2/graviton/]
      --aws-nodes-spot            nodes spot on AWS EKS compute nodes
      --bot-password string       initial password to use while establishing the bot account
      --clean                     delete any local kubefirst content ~/.kubefirst, ~/.k1
      --cloud string              the cloud to provision infrastructure in (default "k3d")
      --cluster-name string       the cluster name, used to identify resources on cloud provider (default "kubefirst")
  -c, --config string             File to be imported to bootstrap configs
      --dry-run                   set to dry-run mode, no changes done on cloud provider selected
      --experimental-mode         whether to allow experimental behavior or developer mode of installer,
                                    not recommended for most use cases, as it may mix versions and create unexpected behavior.
      --git-provider string       specify "github" or "gitlab" git provider. defaults to github. (default "github")
      --github-host string        Github URL (default "github.com")
      --github-owner string       Github owner of repos
      --github-user string        Github user
      --gitops-branch string      version/branch used on git clone - former: version-gitops flag
      --gitops-owner string       git owner of gitops, this may be a user or a org to support forks for testing (default "kubefirst")
      --gitops-repo string        version/branch used on git clone (default "gitops")
  -h, --help                      help for init
      --hosted-zone-name string   the domain to provision the kubefirst platform in
      --metaphor-branch string    version/branch used on git clone - former: version-gitops flag
      --profile string            AWS profile located at ~/.aws/config
      --region string             the region to provision the cloud resources in
      --s3-suffix string          unique identifier for s3 buckets
      --silent                    enable silent mode will make the UI return less content to the screen
      --skip-metaphor-services    whether to skip the deployment of metaphor micro-services demo applications
      --template-tag string       fallback tag used on git clone.
                                    Details: if "gitops-branch" is provided, branch("gitops-branch") has precedence and installer will attempt to clone branch("gitops-branch") first,
                                    if it fails, then fallback it will attempt to clone the tag provided at "template-tag" flag (default "development")
      --use-telemetry             installer won't send telemetry data if --use-telemetry=false is set (default true)
```

### cluster create

Cluster Level operations like **create a new cloud cluster** provisioned with all Kubefirst goodies.

```bash
Based on Kubefirst init command, that creates the Kubefirst configuration file, this command start the
cluster provisioning process spinning up the services, and validates the liveness of the provisioned services.

Usage:
  kubefirst cluster create [flags]

Flags:
  -c, --config string    File to be imported to bootstrap configs
      --destroy          destroy resources
      --dry-run          set to dry-run mode, no changes done on cloud provider selected
      --enable-console   If hand-off screen will be presented on a browser UI (default true)
  -h, --help             help for create
      --silent           enable silent mode will make the UI return less content to the screen
      --skip-gitlab      Skip GitLab lab install and vault setup
      --skip-vault       Skip post-gitClient lab install and vault setup
      --use-telemetry    installer won't send telemetry data if --use-telemetry=false is set (default true)
```

### cluster destroy

Destroy the Kubefirst management cluster and all of the components in Kubernetes.

Optional: Skip GitLab Terraform if the registry has already been deleted.

```bash
destroy all the resources installed via Kubefirst installer

Usage:
  kubefirst cluster destroy [flags]

Flags:
  -c, --config string           File to be imported to bootstrap configs
      --dry-run                 set to dry-run mode, no changes done on cloud provider selected
  -h, --help                    help for destroy
      --hosted-zone-delete      delete full hosted zone, use --keep-base-hosted-zone in combination to keep base DNS records (NS, SOA, liveness)
      --hosted-zone-keep-base   keeps base DNS records (NS, SOA and liveness TXT), and delete all other DNS records. Use it in combination with --hosted-zone-delete
      --silent                  enable silent mode will make the UI return less content to the screen
      --skip-base-terraform     whether to skip the terraform destroy against base install - note: if you already deleted registry it doesnt exist
      --skip-delete-register    whether to skip deletion of register application
      --skip-github-terraform   whether to skip the terraform destroy against github - note: if you already deleted registry it doesnt exist
      --skip-gitlab-terraform   whether to skip the terraform destroy against gitlab - note: if you already deleted registry it doesnt exist
      --use-telemetry           installer won't send telemetry data if --use-telemetry=false is set (default true) 
```

### backupSSL
This command creates a backup of secrets from cert manager certificates to a bucket named Kubefirst-<DOMAIN> where secrets can be 
used on the provisioning phase with the flag --recycle-ssl.

```bash
This command create a backupt of secrets from certmanager certificates to bucket named k1-<DOMAIN>
where are using on provisioning phase with the flag

Usage:
  kubefirst backupSSL [flags]

Flags:
  -h, --help               help for backupSSL
      --include-metaphor   Include Metaphor Apps in process (default true)
```

### restoreSSL

Restore the backup SSL that is stored in the S3 bucket and avoid re-issuing new certificates.

```
Command used to restore existing saved to recycle certificates on a newer re-installation on an already used domain.

Usage:
  kubefirst restoreSSL [flags]

Flags:
  -c, --config string   File to be imported to bootstrap configs
      --dry-run         set to dry-run mode, no changes done on cloud provider selected
  -h, --help            help for restoreSSL
      --silent          enable silent mode will make the UI return less content to the screen
      --use-telemetry   installer won't send telemetry data if --use-telemetry=false is set (default true)
```

### state

Kubefirst configuration can be handed over to another user by pushing the Kubefirst config files to a S3 bucket.

```
Kubefirst configuration can be handed over to another user by pushing the Kubefirst config files to a S3 bucket.

Usage:
  kubefirst state [flags]

Flags:
      --bucket-name string   set the bucket name to store the Kubefirst config file
  -h, --help                 help for state
      --pull                 pull Kubefirst config file to the S3 bucket
      --push                 push Kubefirst config file to the S3 bucket
      --region string        set S3 bucket region
```


### local 

Create a local cluster.

```bash 
Kubefirst localhost enable a localhost installation without the requirement of a cloud provider.

Usage:
  kubefirst local [flags]
  kubefirst local [command]

Available Commands:
  destroy     Destroy Kubefirst local cluster

Flags:
      --admin-email string       the email address for the administrator as well as for lets-encrypt certificate emails
      --dry-run                  set to dry-run mode, no changes done on cloud provider selected
      --enable-console           If hand-off screen will be presented on a browser UI (default true)
      --gitops-branch string     version/branch used on git clone
      --gitops-org string        Helpful when using forks of gitops for testing (default "kubefirst")
      --gitops-repo string       Prefix of the repo for gitops template, repo name has -template (default "gitops")
  -h, --help                     help for local
      --log-level string         available log levels are: trace, debug, info, warning, error, fatal, panic (default "info")
      --metaphor-branch string   metaphor application branch
      --silent                   enable silentMode mode will make the UI return less content to the screen
      --skip-metaphor            If metaphor application suite must be skiped to deploy
      --template-tag string      when running a built version, and ldflag is set for the Kubefirst version, it will use this tag value to clone the templates (gitops and metaphor's)
      --use-telemetry            installer won't send telemetry data if --use-telemetry=false is set (default true)

Use "kubefirst local [command] --help" for more information about a command.
```

### local destroy 

Destroys a local cluster.


```bash 
Destroy all the resources installed via Kubefirst local installer

Usage:
  kubefirst local destroy [flags]

Flags:
      --dry-run   set to dry-run mode, no changes done on cloud provider selected
  -h, --help      help for destroy
      --silent    enable silentMode mode will make the UI return less content to the screen
```
