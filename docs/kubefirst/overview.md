# What is Kubefirst?

Kubefirst is a fully automated and operational open source platform that includes some of the most popular open source tools available in the 
Kubernetes space, all working together from a single command. 

We support local, AWS, and Civo clouds. By running our cli commands against your empty environment, you'll get a GitOps cloud management and application delivery ecosystem complete with automated 
Terraform workflows, Vault secrets management, GitLab or GitHub integrations with Argo, and demo applications 
that demonstrate how it all pieces together.

![](../img/kubefirst/kubefirst-arch.png)

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

**Kubefirst dependencies**: brew install will download [AWS IAM Authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) dependency, that is Helm requirement to authenticate to EKS cluster.

## Kubefirst Usage

[//]: # (todo: update wording)
- The `kubefirst` CLI runs on your localhost and will create an **AWS EKS cluster** that includes **GitLab** or **GitHub**, **Vault**, **ArgoCD** and **Argo Workflow**, and example applications with the **Metaphor** application to demonstrate how everything on the platform works.
- The install takes about 30 minutes to execute. Day-2 operations can't commonly be done within the same hour of cluster provisioning. Kubefirst is solving this on our open source platform. We really hope that is worth a [GitHub Kubefirst repository](https://github.com/kubefirst/kubefirst) star to you (top right corner).
- Your self-hosted GitLab or cloud GitHub will come preconfigured with four Git repositories `kubefirst/gitops` and `kubefirst/metaphor-frontend`, `kubefirst/metaphor-go`, `kubefirst/metaphor`.
- All of the infrastructure as code will be in your GitOps repository in the Terraform directory. IAC workflows are fully automated with Atlantis by merely opening a merge request against the `gitops` repository.
- All of the applications running in your Kubernetes cluster are registered in the GitOps repository in the root `/registry` directory.
- The Metaphor repositories only needs an update to the main branch to deliver the example application to your new development, staging, and production environments. It will hook into your new Vault for secrets, demonstrate automated certs, automated DNS, and GitOps application delivery. Our CI/CD is powered by Argo CD, Argo Workflows, GitLab or GitHub, and Vault.
- The result will be the most comprehensive start to managing a Kubernetes-centric cloud entirely on open source that you keep and can adjust as you see fit. It is an exceptional fully functioning starting point, with the most comprehensive scope we've ever seen in open source.
- We'd love to advise your project on next steps - see our available white glove and commercial services.

_Note: If you run in `cloud aws`, this infrastructure will run in your AWS cloud and is subject to associated AWS fees - about $10/day USD. 
to run. Removal of this infrastructure is also automated with a single `kubefirst cluster destroy` command._

## Differences between selection available

|   | local | aws+github | aws+gitlab | civo+github |
|:--|:--:|:--:|:--:|:--:|
|how to use| `kubefirst local` | `kubefirst init --cloud aws` | `kubefirst init --cloud aws --git-provider gitlab` | `kubefirst civo create` |
|argocd| yes | yes | yes | yes |
|argo workflows| yes | yes | yes | yes |
|vault| yes, in dev mode | yes, backed with DynamoDB and KMS| yes, backed with DynamoDB and KMS| yes, in dev mode | 
|atlantis| yes*1 | yes | yes |  yes | 
|metaphor | metaphor-frontend | metaphor suite | metaphor suite| metaphor-frontend | 
|chartmuseum | yes | yes | yes | yes | 
|self-hosted runner| github action runner runner | github action runner runner | gitlab-runner | github action runner runner | 
|HTTPS/SSL Certificates| mkcert| let's encrypt | let's encrypt | let's encrypt |
|external-secrets-operator | yes | yes | yes |  yes | 
|kubefirst console| yes | yes | yes| yes | 
|oidc | no | yes | yes| yes | 

****1: On local, atlantis uses an ngrok tunnel to allow github to call us back, so it may not be production ready.***

****2: Learn more about mkcert [here](./local/install.md#super-powers-user-needs-and-certificates-to-deal-with-https-locally)***

## Console UI

### AWS or Civo Console UI
Once you run `kubefirst cluster create` command at the end of the installation will open a new browser tab with the Console UI at
`https://kubefirst.<your.domain>` to provide you a dashboard to navigate through the different services that were previsioned.

![console ui](../img/kubefirst/github/console.png)

### Local
Once you run `kubefirst local` command at the end of the installation will open a new browser tab with the Console UI at
`https://kubefirst.localdev.me` to provide you a dashboard to navigate through the different services that were previsioned.


## Destroying

Each kubefirst provisioning command also comes with a corresponding destroy to make it easy to destroy any previsioned infrastructure. Specific destroy guidance is provided for each kubefirst platform.

## Learn more

- Learn more about the kubefirst platform tools [here](../explore/overview.md)
- Explore the local install [here](./local/install.md)

## Credit

[Open source projects used on Kubefirst](./credit.md)
