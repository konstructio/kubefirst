# Security

Kubefirst runs against your public cloud provider, and your personal cloud credentials are leveraged in order to conduct the provisioning
of the Kubefirst platform. We have a tremendous respect when it comes to your personal Cloud provider information and we are very careful about 
leveraging these credentials. We do not embed these personal cloud credentials anywhere in the Kubefirst platform that gets provisioned.

## Assume Role
Kubefirst also provides an approach to run the previsioning process on less privileges strategy via 
[AWS Assume Role](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html). The idea is to basically 
configure a AWS role with the permissions you would like to provide to the Kubefirst installer and provide that role to
the [Assume Role Kubefirst command](../tooling/kubefirst-cli.md) via the `init` command.

## TLS Encryption for Ingressed Services
We use different approaches to close any possible attack surface using different technologies and strategies. On the 
service side, we have Vault to store and encrypt your sensitive data. Every resource that is exposed to the outside 
world is encrypted using SSL/TLS via Let's Encrypt.

## Granular Kubernetes Service Accounts with Explicit IAM Roles for Cloud Access
Each of our platform services has the potential to require access to cloud resources to take advantage of artifact storage, database access, 
kms encryption, or things of that nature. Each service account on the platform comes with a dedicated least privilege IAM policy to grant
granular and controlled access to cloud resources on the platform.

## Additional Layers of Security
GitLab, Vault, Atlantis, and External Secrets Operator have had additional security measures implemented in accordance with the respective applications own security guidelines. Each of these have been implemented to provide reasonable starting points on top of a solid security posture for your core application dependencies.