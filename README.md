# Kubefirst CLI

The Kubefirst CLI is a cloud provisioning tool. With simple setup and two CLI commands, we create a kubernetes cluster managed with automated Infrastructure as Code, GitOps asset management and application delivery, secrets management, a sample application delivered to development, staging, and production, and so much more. It's an open source platform ready to be customized to suit your company's needs.

- [DNS Setup](#dns-setup)
- [Clone the Repository](#clone-the-repository)
- [Start the Container](#start-the-container)
- [Connect to the Container](#connect-to-the-container)
- [Initialization](#initialization)
- [Creation](#creation)
- [Access ArgoCD](#access-argocd)
- [Destroy](#destroy)
- [Available Commands](#available-commands)

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

## Clone the repository

Clone the repository to have the latest `main` branch content

```bash
# via HTTPS
git clone https://github.com/kubefirst/kubefirst.git

# via SSH
git clone git@github.com:kubefirst/kubefirst.git
```

## Start the Container

We run everything in isolation with Docker, for that, start the container with:

```bash
docker-compose up kubefirst
```

## Connect to the Container

Open a new terminal to connect to the container to run kubefirst

```bash
docker exec -it kubefirst bash
```

## Initialization

Some process requires previous initialization, for that, run:

```bash
kubefirst init \
--cloud aws \
--profile default \
--region eu-central-1 \
--admin-email user@example.com \
--cluster-name your_cluster_name \
--hosted-zone-name domain.example
```

## Creation

At this point, everything is ready to start provisioning the cloud services, and for that run:

```bash
kubefirst cluster create
```

## Access ArgoCD

```bash
aws eks update-kubeconfig --name your_cluster_name
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
kubectl -n argocd port-forward svc/argocd-server 8080:80
```

## Destroy

It will destroy the kubefirst management cluster, and clean up every change made in the cloud.

```bash
kubefirst destroy
```

## Available Commands

Kubefirst provides extra tooling for handling the provisioning work.

| Command        | Description                                               |
|:---------------|:----------------------------------------------------------|
| argocdSync     | Request ArgoCD to synchronize applications                |
| checktools     | use to check compatibility of .kubefirst/tools            |
| clean          | removes all kubefirst resources locally for new execution |
| cluster create | create a kubefirst management cluster                     |
| destroy        | destroy the kubefirst management cluster                  |
| info           | provides general Kubefirst setup data                     |
| init           | initialize your local machine to execute `create`         |
| version        | print the version number for kubefirst-cli"               |

---

## The Provisioning Process

![kubefirst provisioning diagram](/images/provisioning.png)
