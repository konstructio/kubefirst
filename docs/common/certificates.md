# SSL Certificates

Kubefirst takes advantage of a variety of security layers to make sure the platform is safe and secure for our users. One
important and significant layer is the SSL certificates Kubefirst leverages to protect HTTPS requests. 
[Let's Encrypt](https://letsencrypt.org) is the certificate authority used to manage the certificates used for our
ingressed services.

During the installation process, ArgoCD applications are instantiated, applying the application manifest, and one of the
manifest steps trigger the creation and management of the SSL certificates using Let's Encrypt. Let's Encrypt then 
initializes a challenge and, after the challenge is solved, the SSL certificate is confirmed via Let's Encrypt as a valid
certificate.

## Backup and Restore Certificates

Certificates requests for certs provided by LetsEncrypt are rate limited, and Kubefirst can also help in that regard! We have two features to 
[backupSSL](../tooling/kubefirst-cli.md) and [restoreSSL](../tooling/kubefirst-cli.md) certificates. When a new 
installation is started, one of the functionalities is to backup your SSL certificates at your AWS account on an S3 
bucket following with `k1 + your-domain-name`.

**Practical use case scenario:**

1. You finished the Kubefirst installation.
2. You destroyed the Kubefirst installation.
3. You want to start a fresh installation but want to use the previously created certificates.
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
