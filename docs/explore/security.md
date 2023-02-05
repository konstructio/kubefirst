# Security

## Installation Account
Kubefirst runs against your public cloud provider and leverages your personal cloud credentials in order to conduct the provisioning of the Kubefirst platform. We do not embed your credentials into the platform in any way, they are only used during the installation process.

## Granular Kubernetes Service Accounts with Explicit IAM Roles for Cloud Access
Each of our platform services has the potential to require access to cloud resources to take advantage of artifact storage, database access, 
kms encryption, or things of that nature. Each service account on the platform comes with a dedicated least privilege IAM policy to grant
granular and controlled access to cloud resources on the platform.

## Additional Layers of Security
GitLab, Vault, Atlantis, and External Secrets Operator have had additional security measures implemented in accordance with the respective applications own security guidelines. Each of these have been implemented to provide reasonable starting points on top of a solid security posture for your core application dependencies.

