# Gitlab Repositories

You'll start out in your gitlab server with a couple new gitlab repositories under the kubefirst project.

![](../img/kubefirst/gitlab/kubefirst-group.png)

## Repository Summary

`gitops`
The gitops repo houses all of our IAC and all our gitops configurations. All of the infrastructure that you receive with kubefirst was produced by some combination of terraform and argocd. You will add to this repository as your business needs require new infrastructure or new applications.

`metaphor`
Metaphor is an example application used to showcase certain features of the kubefirst platform. Metaphor has CI/CD that delivers the app to a development, staging, and production namespace in your kubernetes cluster. Its secrets in vault are bound to the metaphor app through the use of external-secrets, a handy kubernetes utility to keep kubernetes secrets in sync with the vault source of truth. It also demonstrates how dns entries automatically will be automtically created in route53 using external-dns. It has auto-renewing short lived certificates generated and auto-renewed as well using cert-manager and the letsencrypt clusterissuer.

## GitLab Repository Management

These gitlab repositories are being managed in terraform.

As you need additional gitlab repositories, just add a new section of terraform code to `terraform/gitlab/kubefirst-repos.tf`
```
module "your_repo_name" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  group_name                            = gitlab_group.kubefirst.id
  repo_name                             = "your-repo-name"
  create_ecr                            = true
  initialize_with_readme                = true
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
}
```

Gitlab's terraform provider provides many more configuration than just these settings. Check them out and add to your default settings once you're comfortable with the platform.

Take a look at the `Resources` section of the gitlab provider documentation [](https://registry.terraform.io/providers/gitlabhq/gitlab/latest/docs/resources).

That was just gitlab. Take a look at all of the terraform providers that are available, the list of technologies you can manage in terraform is really getting impressive. [](https://www.terraform.io/docs/providers/index.html)

## Making Terraform Changes

To make infrastructure and configuration changes with terraform, simply open a merge request in the `gitops` repository. Your merge request will automatically provide plans, state locks, and applies, and even comment in the merge request itself. You'll have a simple, peer reviewable, auditable changelog of all infrastructure and configuration changes.

![](../img/kubefirst/gitlab-repositories/terraform-atlantis-merge-request.png)
