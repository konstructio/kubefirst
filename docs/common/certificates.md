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

At any point after the installation, the user can restore the certificates using 
[restoreSSL](../tooling/kubefirst-cli.md) command.


### How/When to use Kubefirst Certificates tool?

After an successful installation, if possible after `metaphor` applciations we deployed, run the command on your terminal at the folder you executed you installation. 

```bash 
kubefirst backupSSL
```

`kubefirst backupSSL` will create a bucket with the format `k1 + your-domain-name` with the certificates already generated at the moment of the command execution. 

Then on a future execution of `kubefirst cluster create` the installer will automatically call for you `kubefirst restoreSSL` when the cluster is ready to receive certificates and avoid spend your certificates quotas on [Let's Encrypt](https://tools.letsdebug.net/cert-search?m=domain). 


### How I remove my certificates back?

- Go to your aws account 
- Find the s3 bucket: `k1 + your-domain-name`
- Empty it
- Remove it




### Known issues

- If you change you installation region, you will not be able to recycle certificates as the buckets are bound to the same region of the installation backup was done. The workaround is [to download bucket contents, destroy bucket, wait until aws allows to create it again in the new region and repopulate it with the downloaded files](https://github.com/kubefirst/kubefirst/issues/421#issuecomment-1252390475). 
