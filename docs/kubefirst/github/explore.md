# Explore

[//]: # (`todo: need new getting started video for github`)

<div class="video-wrapper">
  <iframe width="1280" height="400" src="https://www.youtube.com/embed/KEUOaNMUqOM" frameborder="0" allowfullscreen></iframe>
</div>

**psssst** *- if you plan to destroy your kubefirst platform and recreate it again we recommend running `kubefirst backupSSL` to re-use your ssl certs from Let's Encrypt. See the [docs](https://docs.kubefirst.io/common/certificates.html#backup-and-restore-certificates).*

The `kubefirst cluster create` execution includes important information toward the end, including URLs and passwords. Please save this information! 

You now have an EKS cluster with the following content installed in it:

| Application                  | Description                                                                |
|------------------------------|----------------------------------------------------------------------------|
| Nginx Ingress Controller     | Ingress Controller                                                         |
| Cert Manager                 | Certificate Automation Utility                                             |
| Certificate Issuers          | Let's Encrypt browser-trusted certificates                                 |
| Argo CD                      | GitOps Continuous Delivery                                                 |
| Argo Workflows               | Application Continuous Integration                                         |
| GitHub Action Runner         | GitHub CI Executor                                                         |
| Vault                        | Secrets Management                                                         |
| Atlantis                     | Terraform Workflow Automation                                              |
| External Secrets             | Syncs Kubernetes secrets with Vault secrets                                |
| Chart Museum                 | Helm Chart Registry                                                        |
| Metaphor JS API              | (development, staging, production) instance of sample application          |
| Metaphor Go API              | (development, staging, production) instance of sample go application       |
| Metaphor Frontend            | (development, staging, production) instance of sample frontend application |

- These apps are all managed by Argo CD and the app configurations are in the `gitops` repo's `registry` folder.
- The AWS infrastructure is terraform - that's also in your `gitops` repo, but in your `terraform` folder.

![](../../img/kubefirst/getting-started/gitops-assets.png)

## Step 1: Console UI

Once you run the `cluster create` command at the end of the installation will open a new browser tab with the Console UI at
`http://localhost:9094` to provide you a dashboard to navigate through the different services that were provisioned.

![console ui](../../img/kubefirst/github/console.png)

![terminal handoff](../../img/kubefirst/getting-started/cluster-create-result.png)

These are **not your personal credentials**. These are administrator credentials that can be used if you ever need to 
authenticate and administer your tools if your OIDC provider ever becomes unavailable. Please protect these secrets and 
store them in a safe place.

## Step 2: Add Your Team(optional)

This step is meant to explore the onboarding process of a new user to your installation:

- [Explore Atlantis & Terraform to manage users](../../common/terraform.html#how-can-i-use-atlantis-to-add-a-new-user-on-my-github-backed-installation)



## Step 3: Deliver Metaphor to Development, Staging, and Production

Metaphor is our sample application that we use to demonstrate parts of the platform and to test CI changes.

If you visit its `/.github/workflows/main.yaml` in the metaphor repo, you'll see it's just sending some workflows to argo in your local EKS cluster. Those workflows are also in the `metaphor` repo in the `.argo` directory.

The metaphor pipeline will:

- Publish the metaphor container to your private ECR.
- add the metaphor image to a release candidate helm chart and publish it to chartmuseum
- set the metaphor with the desired Helm chart version in the GitOps repo for development and staging
- the release stage of the pipeline will republish the chart, this time without the release candidate notation making it an officially released version and prepare the metaphor application chart for the next release version
- the officially released chart will be set as the desired Helm chart for production.

To watch this pipeline occur, make any change to the `main` branch of the `metaphor` repo. If you're not feeling creative, you can just add a newline to the README.md. Once a file in `main` is changed, navigate to metaphor's CI/CD in the github Actions tab to see the workflows get submitted to Argo workflows.

You can visit the metaphor development, staging, and production apps in your browser to see the versions change as you complete resources and ArgoCD syncs the apps. The metaphor URLs can be found in your GitOps and metaphor project `README.md` files.

## Learning the Ropes

We've tried our best to provide the available customizations and patterns of the Kubefirst platform here on our docs site. We've also made [links available](./credit.md) to all of our open source tool's sources of documentation.

You can [reach out to us](../../community/index.md) if you have any issues along the way. We're also available for consultation of where you should take the platform based on your organization's needs. We know the technologies inside and out and would love to help you do the same.

## What to do next

Continue your journey: 

- [Destroying](./destroy.md)
