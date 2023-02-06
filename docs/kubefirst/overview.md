## How to install Kubefirst CLI

```shell
brew install kubefirst/tools/kubefirst
```

There are a few other ways to install Kubefirst for different operating systems, architectures, and containerized environments. See our [installation README](https://github.com/kubefirst/kubefirst/blob/main/build/README.md) for non-brew details.

To upgrade an existing Kubefirst install to the latest version run

```shell
brew update
brew upgrade kubefirst
```

## Kubefirst Usage

[//]: # (todo: update wording)
- The `kubefirst` CLI runs on your localhost and will create an GitLab or GitHub kubernetes ecosystem including Vault, ArgoCD, Argo Workflows, self-hosted runners for GitHub and GitLab, and an application to demonstrate how everything on the platform works.
- We have local, AWS, and Civo platforms available.
- The install takes about 30 minutes to execute.
- Your self-hosted GitLab or SaaS GitHub will come with a `gitops` and `metaphor` repository 
- All of the infrastructure as code will be in your GitOps repository in the Terraform directory. IAC workflows are fully automated with Atlantis by merely opening a merge request against the `gitops` repository.
- All of the applications running in your Kubernetes cluster are registered in the gitops repository in the root `/registry` directory.
- The metaphor repositories only needs an update to the main branch to deliver the example application to your new development, staging, and production environments. It will hook into your new Vault for secrets, demonstrate automated certs, automated DNS, and GitOps application delivery. Our CI/CD is powered by Argo CD, Argo Workflows, GitLab or GitHub, and Vault.

## Platforms

|   | local | aws + github | aws + gitlab | civo + github |
|:--|:--:|:--:|:--:|:--:|
|how to use | `kubefirst local` | `kubefirst init --cloud aws` | `kubefirst init --cloud aws --git-provider gitlab` | `kubefirst civo create` |
|argocd | yes | yes | yes | yes |
|argo workflows| yes | yes | yes | yes |
|vault | yes, in dev mode | yes, backed with DynamoDB and KMS| yes, backed with DynamoDB and KMS| yes, in dev mode | 
|atlantis | yes *1 | yes | yes |  yes | 
|metaphor | metaphor-frontend | metaphor suite | metaphor suite| metaphor-frontend | 
|chartmuseum | yes | yes | yes | yes | 
|self-hosted runner| github action runner runner | github action runner runner | gitlab-runner | github action runner runner | 
|HTTPS/SSL Certificates | mkcert| let's encrypt | let's encrypt | let's encrypt |
|external-secrets-operator | yes | yes | yes |  yes | 
|kubefirst console| yes | yes | yes| yes | 
|oidc | no | yes | yes | yes | 

*1: On local, atlantis uses an ngrok tunnel to allow github to call us back, so it may not be production ready.*

## Console UI

### AWS or Civo Console UI
Once you run `kubefirst cluster create` command at the end of the installation will open a new browser tab with the Console UI at
`https://kubefirst.<your.domain>` to provide you a dashboard to navigate through the different services that were previsioned.

![console ui](../img/kubefirst/github/console.png)

### Local Console UI
Once you run `kubefirst local` command at the end of the installation will open a new browser tab with the Console UI at
`https://kubefirst.localdev.me` to provide you a dashboard to navigate through the different services that were previsioned.

## Destroying

Each kubefirst provisioning command also comes with a corresponding destroy to make it easy to destroy any previsioned infrastructure. Specific destroy guidance is provided for each kubefirst platform.

## Learn more

- Learn more about the kubefirst platform tools [here](../explore/overview.md)
- Explore the local install [here](./local/install.md)

## Credit

[Open source projects used on Kubefirst](./credit.md)
