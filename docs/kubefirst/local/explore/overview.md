# Explore

[//]: # (`todo: need new getting started video for github local`)

[//]: # (<iframe width="784" height="441" src="https://www.youtube.com/embed/KEUOaNMUqOM" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>)

`kubefirst local` provides URLs and passwords that point to local applications. These applications are hosted using [k3d](https://k3d.io) a lightweight wrapper to run k3s (Rancher Lab’s minimal Kubernetes distribution)

If you close the handoff screen (by pressing ESC), you can still access the [Kubefirst Console](https://kubefirst.localdev.me) to see all applications, and their local endpoints by opening the Console app.

A newly provisoned local Kubefirst cluster contains the following content installed in it:

| Application                  | Description                                                                 |
|------------------------------|-----------------------------------------------------------------------------|
| Traefik Ingress Controller   | Native k3d Ingress Controller                                               |
| Cert Manager                 | Certificate Automation Utility                                              |
| Argo CD                      | GitOps Continuous Delivery                                                  |
| Argo Workflows               | Application Continuous Integration                                          |
| GitHub Action Runner         | GitHub CI Executor                                                          |
| Vault                        | Secrets Management                                                          |
| Atlantis                     | Terraform Workflow Automation                                               |
| External Secrets             | Syncs Kubernetes secrets with Vault secrets                                 |
| Chart Museum                 | Helm Chart Registry                                                         |
| Metaphor Frontend            | (development, staging, production) instance of sample Next.js and React app |

- These apps are all managed by Argo CD and the app configurations are in the `gitops` repo's `registry` folder.

## Introduction to the Console UI

[Learn how to use the Kubefirst Console](../console.md)

## Atlantis Example / User Creation Walkthrough

[Create your own user through Atlantis](../user-creation.md)

## Deliver `metaphor-frontend` to your new Development, Staging, and Production Environments

[Use metaphor as a sample frontend](../metaphor.md)

## Accessing the Kubefirst Applications

The Console UI will provide you with the URLs to access the applications that were provisioned, and the handoff screen will provide the credentials to login into these applications.

After closing the handoff screen, you can also have access to the same credentials via the `~/.kubefirst` file, that hosts the initial credentials for all the installed applications.

## Learning the Ropes

We've tried our best to provide the available customizations and patterns of the Kubefirst platform here on our docs site. We've also made [links available](../../credit.md) to all of our open source tool's sources of documentation.

You can [reach out to us](../../../community/index.md) if you have any issues along the way. We're also available for consultation of where you should take the platform based on your organization's needs. We know the technologies inside and out and would love to help you do the same.