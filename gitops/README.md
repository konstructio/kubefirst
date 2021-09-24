![](logo.png)

# gitops

The `gitops` repository has 2 main section

- `/registry`: the argocd gitops app registry 
- `/terraform`: infrastructure as code & configuration as code

## kubefirst apps

The [kubefirst/nebulous](https://hub.docker.com/repository/docker/kubefirst/nebulous) installation has established the following applications:

| Application              | Namespace        | Description                                 | URL (where applicable)                              |
|--------------------------|------------------|---------------------------------------------|-----------------------------------------------------|
| GitLab                   |                  | Privately Hosted GitLab Omnibus Server      | https://gitlab.<AWS_HOSTED_ZONE_NAME>               |
| Vault                    | vault            | Secrets Management                          | https://vault.<AWS_HOSTED_ZONE_NAME>                |
| Argo CD                  | argocd           | GitOps Continuous Delivery                  | https://argocd.<AWS_HOSTED_ZONE_NAME>               |
| Argo Workflows           | argo             | Application Continuous Integration          | https://argo.<AWS_HOSTED_ZONE_NAME>                 |
| Keycloak                 | keycloak         | Authentication                              | https://keycloak.<AWS_HOSTED_ZONE_NAME>             |
| Atlantis                 | atlantis         | Terraform Workflow Automation               | https://atlantis.<AWS_HOSTED_ZONE_NAME>             |
| Chart Museum             | chartmuseum      | Helm Chart Registry                         | https://chartmuseum.<AWS_HOSTED_ZONE_NAME>          |
| Metaphor Development     | development      | Development instance of sample application  | https://metaphor-development.<AWS_HOSTED_ZONE_NAME> |
| Metaphor Staging         | staging          | Staging instance of sample application      | https://metaphor-staging.<AWS_HOSTED_ZONE_NAME>     |
| Metaphor Production      | production       | Production instance of sample application   | https://metaphor-production.<AWS_HOSTED_ZONE_NAME>  |
| Nginx Ingress Controller | ingress-nginx    | Ingress Controller                          |                                                     |
| Cert Manager             | cert-manager     | Certificate Automation Utility              |                                                     |
| Certificate Issuers      | clusterwide      | Let's Encrypt browser-trusted certificates  |                                                     |
| External Secrets         | external-secrets | Syncs Kubernetes secrets with Vault secrets |                                                     |
| GitLab Runner            | gitlab-runner    | GitLab CI Executor                          |                                                     |

## argocd registry

The argocd configurations in this repo can be found in the [registry directory](./registry). The applications that you build and release on the kubefirst platform will also be registered here in the development, staging, and production folders. The `metaphor` app can be found there to serve as an example to follow.

The `main` branch of this repo represents the desired state all apps registered with kubernetes. Argo CD will automatically try to converge your desired state with the actual state in kubernetes with a process called Argo Sync. You can see the Sync status of all of your apps in the [argo cd ui](https://argo.<AWS_HOSTED_ZONE_NAME>).

## terraform infrastructure as code

The terraform in this repository can be found in the `/terraform` directory. 

All of our terraform is automated with atlantis. To see the terraform entry points and under what circumstance they are triggered, see [atlantis.yaml](./atlantis.yaml).

Any change to a `*.tf` file, even a whitespace change, will trigger its corresponding Atlantis workflow once a merge request is submitted in GitLab. Within a minute it will post the plan to the pull request with instruction on how to apply the plan if approved.

## terraform configuration as code

In addition to infrastructure terraform, the `gitops` repository also contains configuration as code for the following products:
- ArgoCD: The Argo CD app-registry, repositories, and secrets
- GitLab: Gitlab Repositories and ECR registries needed to house containers for those repositories
- Keycloak: Kubefirst Realm, User Groups for Admin and Developer roles, and sample users that get SSO access to vault, argo cd, argo workflows, and gitlab
- Vault: auth backends, secrets engine, infrastructure secrets

## engineering onboarding

Your kubefirst platform comes with some terraform in place for managing [admins](./terraform/keycloak/admins.tf) and [developers](./terraform/keycloak/developers.tf). At the top of these two files, you'll find a list of sample admins and developers. Replace this list with the list of actual users you want added to the admin and developer groups and open a pull request. The pull request will show you the user changes in the terraform plan. When approved, have atlantis apply the plan with an `atlantis apply` comment in the pull request.

Your new users will have temporary passwords generated for them and stored in Vault in the `/users` secret store.

