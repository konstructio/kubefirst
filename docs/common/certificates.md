# SSL Certificates

Kubefirst takes advance of a variety of security layers to make the platform safe and secure for our users. One
important and significant layer are the SSL certificates Kubefirst make use to protect https requests. 
[Let's Encrypt](https://letsencrypt.org) is the chose tool to manage the services installed via Kubefirst installation.

During the installation process, ArgoCD applications are instantiated, applying the application manifest, and one of the
manifest steps trigger the creation and management of the SSL certificates using Let's Encrypt. Let's encrypt than 
initialize a challenge, and after the challenge is solved, the SSL certificate is confirmed via Let's Encrypt as a valid
certificate, if not, ArgoCD will use his power to do it's best to resolve the certificate challenge.

## Backup and Restore certificates

Certificates are limited, and Kubefirst can also help on that regard! We have two features to 
[backupSSL](../tooling/kubefirst-cli.md) and [restoreSSL](../tooling/kubefirst-cli.md) certificates. When a new 
installation is started, one of the functionalities is to backup your SSL certificates at your AWS account on a S3 
bucket following with `k1 + your-cluster-name`.

At any point after the installation, the user can restore the certificates using 
[restoreSSL](../tooling/kubefirst-cli.md) command.

**Practical use case scenario:**

1. user finish Kubefirst installation
2. user destroyed the Kubefirst installation
3. user wants to start a fresh new installation but wants to use the previously created certificates
4. user call this sequence of commands: `kubefirst clean`, `kubefirst init + <args>`, `kubefirst restoreSSL`, and `kubefirst cluster create`
