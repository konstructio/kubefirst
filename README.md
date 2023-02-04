<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="images/kubefirst-light.svg" alt="Kubefirst Logo">
    <img alt="" src="images/kubefirst.svg">
  </picture>
</p>
<p align="center">
  GitOps Infrastructure & Application Delivery Platform
</p>

<p align="center">
  <a href="https://docs.kubefirst.io/">Install</a>&nbsp;|&nbsp;
  <a href="https://twitter.com/kubefirst">Twitter</a>&nbsp;|&nbsp;
  <a href="https://www.linkedin.com/company/kubefirst">LinkedIn</a>&nbsp;|&nbsp;
  <a href="https://join.slack.com/t/kubefirst/shared_invite/zt-r0r9cfts-OVnH0ooELDLm9n9p2aU7fw">Slack</a>&nbsp;|&nbsp;
  <a href="https://kubeshop.io/blog-projects/kubefirst">Blog</a>
</p>

<p align="center">
  <a href="https://github.com/kubefirst/kubefirst/releases"><img title="Release" src="https://img.shields.io/github/v/release/kubefirst/kubefirst"/></a>
  <!-- <a href=""><img title="Docker builds" src="https://img.shields.io/docker/automated/kubeshop/tracetest"/></a> -->
  <a href="https://github.com/kubefirst/kubefirst/releases"><img title="Release date" src="https://img.shields.io/github/release-date/kubefirst/kubefirst"/></a>
</p>


---

# Kubefirst CLI

The Kubefirst CLI is a cloud provisioning tool. With simple setup and two CLI commands, we create a kubernetes cluster 
managed with automated Infrastructure as Code, GitOps asset management and application delivery, secrets management, a 
sample application delivered to development, staging, and production, and so much more. It's an open source platform 
ready to be customized to suit your company's needs.

- [Kubefirst CLI](#kubefirst-cli)
  - [DNS Setup](#dns-setup)
  - [Installing the CLI](#installing-the-cli)
  - [Other installation techniques:](#other-installation-techniques)
  - [Initialization](#initialization)
    - [localhost](#localhost)
    - [GitLab](#gitlab)
    - [GitHub](#github)
  - [Creation](#creation)
  - [Destroy](#destroy)
- [What to do next](#what-to-do-next)
- [If you want learn more](#if-you-want-learn-more)
  - [Access ArgoCD](#access-argocd)
  - [Available Commands](#available-commands)
  - [The Provisioning Process](#the-provisioning-process)
  - [Feed K-Ray](#feed-k-ray)

![kubefirst architecture diagram](/images/kubefirst-arch.png)

## DNS Setup

In order to install Kubefirst it's required to have a public domain. For root domains, setting the `--hosted-zone-name`
is enough, in case you want to use subdomains, and the domain is hosted on AWS, please follow the
[AWS documentation](https://aws.amazon.com/premiumsupport/knowledge-center/create-subdomain-route-53/).

Provisioned services on root domain will be hosted as:

```bash
argocd.example.com
gitlab.example.com
...
```

Provisioned services on subdomains will be hosted as:

```bash
argocd.subdomain.example.com
gitlab.subdomain.example.com
...
```

## Installing the CLI

```bash 
brew install kubefirst/tools/kubefirst
```

[Learn more](https://docs.kubefirst.com/kubefirst/overview.html)

## Other installation techniques:

[Details Here](./build/README.md)

## Initialization

Since Kubefirst 1.9 version, GitHub is also available as a Git platform provider alongside GitLab.


### localhost

localhost enables Kubefirst to be installed in your local machine, not requiring an AWS account, using localhost you can:

```bash
kubefirst local
```

### GitLab

To prepare the installation using GitLab you can:

```bash
kubefirst init \
--admin-email yourname@yourcompany.com \
--cloud aws \
--hosted-zone-name yourdomain.com \
--region us-east-1 \
--profile default \
--cluster-name your_cluster_name \
--s3-suffix you-s3-bucket-name \
--git-provider gitlab \
--aws-nodes-spot
```

### GitHub

To prepare the installation using GitHub you can:

```bash
export KUBEFIRST_GITHUB_AUTH_TOKEN=your_github_auth_token

kubefirst init \
--admin-email yourname@yourcompany.com \
--cloud aws \
--hosted-zone-name yourdomain.com \
--region us-east-1 \
--profile default \
--cluster-name your-cluster-name \
--github-owner your-github-organization-name 

```

## Creation

At this point, everything is ready to start provisioning the cloud services, and for that run:

```bash
kubefirst cluster create
```

## Destroy

It will destroy the kubefirst management cluster, and clean up every change made in the cloud.

```bash
kubefirst cluster destroy
```

or for `kubefirst local`:

```bash
kubefirst local destroy
```

# What to do next

[Learn More - Getting Started](https://docs.kubefirst.com/index.html)


# If you want learn more 

## Access ArgoCD

```bash
aws eks update-kubeconfig --name your_cluster_name
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
kubectl -n argocd port-forward svc/argocd-server 8080:80
```

## Available Commands

- [Kubefirst Commands](https://docs.kubefirst.com/tooling/kubefirst-cli.html)


## The Provisioning Process

![kubefirst provisioning diagram](/images/provisioning.png)

## Feed K-Ray

Did you know our superhero mascot K-Ray gets its frictionless superpowers from a healthy diet of GitHub stars? K-Ray gets soooo hungry too - you wouldn't believe it. Feed K-Ray a GitHub star ⭐ above to bookmark our project and keep K-Ray happy!!

[![Star History Chart](https://api.star-history.com/svg?repos=kubefirst/kubefirst&type=Date)](https://star-history.com/#kubefirst/kubefirst&Date)
