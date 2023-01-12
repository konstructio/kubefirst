# After Install

[//]: # (`todo: need new getting started video for github local`)

[//]: # (<iframe width="784" height="441" src="https://www.youtube.com/embed/KEUOaNMUqOM" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>)

The `kubefirst local` execution includes important information toward the end, including URLs and passwords to get to your applications.

If you close the handoff screen (by pressing ESC), you can still access the Kubefirst Console to see all applications, and their local endpoints by opening the Console app.

You now have a k3d cluster with the following content installed in it:

| Application                  | Description                                                                            |
|------------------------------|----------------------------------------------------------------------------------------|
| Traefik Ingress Controller   | Native k3d Ingress Controller                                                          |
| Cert Manager                 | Certificate Automation Utility                                                         |
| Argo CD                      | GitOps Continuous Delivery                                                             |
| Argo Workflows               | Application Continuous Integration                                                     |
| GitHub Action Runner         | GitHub CI Executor                                                                     |
| Vault                        | Secrets Management                                                                     |
| Atlantis                     | Terraform Workflow Automation                                                          |
| External Secrets             | Syncs Kubernetes secrets with Vault secrets                                            |
| Chart Museum                 | Helm Chart Registry                                                                    |
| Metaphor Frontend            | (development, staging, production) instance of sample Nextjs and React app             |

- These apps are all managed by Argo CD and the app configurations are in the `gitops` repo's `registry` folder.

## Step 1: Console UI

![terminal handoff](../../img/kubefirst/local/console.png)

The `kubefirst local` command will open a new browser tab at completion with the Console UI at
`https://kubefirst.localdev.me` to provide you an easy way to navigate through the different services that were provisioned.

![terminal handoff](../../img/kubefirst/local/handoff-screen.png)

## Step 2: Make your first automated Terraform change

Go to your new gitops repository in your personal GitHub. Navigate to the `gitops` project and edit the file `terraform/users/admins-github.tf`. In this file, you'll see some blocks that represent admin users - the `kubefirst_bot` user, and a commented-out `admin_one` user.


```
module "admin_one" {
  source = "./modules/user/github"

  acl_policies            = ["admin"]
  email                   = "admin@your-company-io.com"
  first_name              = "Admin"
  github_username         = "admin_one_github_username"
  last_name               = "One"
  username                = "aone"
  user_disabled           = false
  userpass_accessor       = data.vault_auth_backend.userpass.accessor
}
```

To exercise the user onboarding process, uncomment that admin_one user. Edit this code to replace the values for the `email`, `first_name`, `github_username`, `last_name`, and `username`. 

With the name of your new module in mind, edit the list of `vault_identity_group_member_entity_ids` at the top of this file, adding your new module to the list.

Commit this change to a **new branch** and create a merge request. This will kick off the Atlantis workflow. Within a minute or so of submitting the merge request, a comment will appear on the merge request that shows the terraform plan with the changes it will be making to your infrastructure. 

To apply these changes, submit a comment on that Merge Request with the following comment text:
```
atlantis apply
```

Doing so will instruct Atlantis to apply the plan. It will report back with the results of the apply within a minute or so.

NOTE: Atlantis merges your Pull Request automatically once an apply is successfully executed. Don't merge Terraform merge requests yourself.

Atlantis will always run plans automatically for you when a merge request is opened that changes files mapped in `atlantis.yaml`

Any new users you have created through this process will have their temporary initial passwords stored in Vault. You can access Vault using the information provided to you in the terminal as well, and you will find your users' individual initial passwords in the Vault secret store `/secrets/users/<username>`.

![](../../img/kubefirst/getting-started/vault-users.png)

Once you've provided them this initial password, they can update their own password throughout the platform by updating their userpass entity in vault. Anyone can change their own password, and Admins can reset anyone's password. These rules, just like everything else on Kubefirst, can be configured in your new gitops repository.

![](https://user-images.githubusercontent.com/53096417/204801723-602beff0-12f9-45a9-bb9c-4d85a889d1ce.gif)

## Step 3: Deliver `metaphor-frontend` to your new Development, Staging, and Production

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

We've tried our best to provide the available customizations and patterns of the Kubefirst platform here on our docs site. We've also made [links available](./credit.md) to all of our open source tool's sources of documentation.

You can [reach out to us](../../community/index.md) if you have any issues along the way. We're also available for consultation of where you should take the platform based on your organization's needs. We know the technologies inside and out and would love to help you do the same.
