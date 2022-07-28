![K-Ray, the Kubefirst mascot](/images/kubefirst.svg)

---

# Kubefirst CLI

Kubefirst CLI is a cloud provisioning tool. With simple setup and few CLI calls, we spin up a full AWS cluster with full
GitOps integration, secrets management, production and development Kubernetes environments ready to be consumed.

- [Environment Variables](#environment-variables)
- [DNS setup](#dns-setup)
- [Start the container](#start-the-container)
- [Initialization](#initialization)
- [Creation](#creation)
- [Access ArgoCD](#access-argocd)
- [Destroy](#destroy)
- [Available Commands]()

![kubefirst architecture diagram](/images/kubefirst-arch.png)

## Environment Variables

The setup is extremely simple, create a `.env` file in the root folder, and add the following variables:

| Variable    | example      |
|-------------|--------------|
| AWS_PROFILE | default      |
| AWS_REGION  | eu-central-1 |

## DNS Setup

...

## Start the container

We run everything on isolation with Docker, for that, start the container with:

```bash
docker-compose up kubefirst-dev
```

## Initialization

Some process requires previous initialization, for that, run:

```bash
mkdir -p ~/.kubefirst
go run . init --admin-email user@example.com --cloud aws --hosted-zone-name domain.example --region eu-central-1 --cluster-name your_cluster_name
```

## Creation

At this point, everything is ready to start provisioning the cloud services, and for that we can run:

```bash
go run . cluster create
```

## Access ArgoCD

```bash
aws eks update-kubeconfig --name kubefirst
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
kubectl -n argocd port-forward svc/argocd-server 8080:80
```

## Destroy

It will destroy the kubefirst management cluster, and clean up every change made in the cloud.

```bash

go run . destroy
rm -rf ~/.kubefirst
rm ~/.flare
```

## Available Commands

Kubefirst provides extra tooling for handling the provisioning work.

| Command    | Description                                               |
|:------------|:-----------------------------------------------------------|
| argocdSync     | Request ArgoCD to synchronize applications                |
| checktools     | use to check compatibility of .kubefirst/tools            |
| clean          | removes all kubefirst resources locally for new execution |
| cluster create | create a kubefirst management cluster                     |
| destroy    | destroy the kubefirst management cluster                  |
| info       | provides general Kubefirst setup data                     |
| init       | initialize your local machine to execute `create`         |
| version    | print the version number for kubefirst-cli"               |

---
## The provisioning process
![kubefirst provisioning diagram](/images/provisioning.png)
