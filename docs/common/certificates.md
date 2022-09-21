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
bucket following with `k1 + your-domain-name`.

**Practical use case scenario:**

1. you finished Kubefirst installation
2. you destroyed the Kubefirst installation
3. you want to start a fresh new installation but wants to use the previously created certificates
4. at this point you can call this sequence of commands: 

```bash
# backup your SSL certificates at S3 bucket name "k1-your-domain-name"
kubefirst backupSSL
# clean previous installation
kubefirst clean
# prepare a new installation
kubefirst init + <args>
# during Kubefirst installation process,
# the installation will load the backup certificates and use it to avoid issuing new certificates.
# no manual changes are necessary for certificate backup restore
kubefirst cluster create`
```
