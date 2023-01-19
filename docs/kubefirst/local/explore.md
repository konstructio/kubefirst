# Explore

[//]: # (`todo: need new getting started video for github local`)

[//]: # (<iframe width="784" height="441" src="https://www.youtube.com/embed/KEUOaNMUqOM" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>)

The `kubefirst local` execution includes important information toward the end, including URLs and passwords to get to your applications.

If you close the handoff screen (by pressing ESC), you can still access the Kubefirst Console to see all applications, and their local endpoints by opening the Console app.

You now have a k3d cluster with the following content installed in it:

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

## Step 1: Console UI

![terminal handoff](../../img/kubefirst/local/console.png)

The `kubefirst local` command will open a new browser tab at completion with the Console UI at
`https://kubefirst.localdev.me` to provide you an easy way to navigate through the different services that were provisioned.

![terminal handoff](../../img/kubefirst/local/handoff-screen.png)

## Step 2: Make your first automated Terraform change(optional)

This step is meant to explore the onboarding process of a new user to your installation:

- [Explore Atlantis & Terraform to manage users](../../common/terraform.html#how-can-i-use-atlantis-to-add-a-new-user-on-my-github-backed-installation)

## Step 3: Accessing the applications

The Console UI will provide you with the URLs to access the applications that were provisioned, and the handoff screen will provide the credentials to login into these applications.

After closing the handoff screen, you can also have access to the same credentials via the `~/.kubefirst` file, that hosts the initial credentials for all the installed applications.

## Step 4: Deliver `metaphor-frontend` to your new Development, Staging, and Production

The `metaphor-frontend` repo is a simple sample microservice with source code, build, and delivery automation that we use to demonstrate parts of the platform. We also find it to be a valuable way to test CI changes without impacting real apps on your platform.

If you visit its `/.github/workflows/main.yaml`, you'll see that it's just sending some workflows to argo workflows in your local k3d cluster.

The example delivery pipeline will:

- Publish the metaphor container to your private github.
- add the metaphor image to a release candidate helm chart and publish it to chartmuseum
- set the metaphor with the desired Helm chart version in the GitOps repo for development and staging
- the release stage of the pipeline will republish the chart, this time without the release candidate notation making it an officially released version and prepare the metaphor application chart for the next release version
- the officially released chart will be set as the desired Helm chart for production.

To watch this pipeline occur, make any change to the `main` branch of of the `metaphor-frontend`. If you're not feeling creative, you can just add a newline to the `README.md`. Once a file in `main` is changed, navigate to metaphor-frontend's CI/CD in the github `Actions` tab to see the workflows get submitted to Argo workflows.

You can visit the metaphor-frontend development, staging, and production apps in your browser to see the versions change as you complete resources and ArgoCD syncs the apps. The metaphor-frontend URLs can be found in your gitops and metaphor-frontend project `README.md` files.

## Learning the Ropes

We've tried our best to provide the available customizations and patterns of the Kubefirst platform here on our docs site. We've also made [links available](../credit.md) to all of our open source tool's sources of documentation.

You can [reach out to us](../../community/index.md) if you have any issues along the way. We're also available for consultation of where you should take the platform based on your organization's needs. We know the technologies inside and out and would love to help you do the same.

## What to do next

Continue your journey: 

- [Destroying](./destroy.md)
