# Terraform and Atlantis
`terraform` is our infrastructure as code layer and we manage our terraform workflows with `atlantis` automation.

## Making Changes In Terraform

### Automatic Plans With Atlantis
Any merge request that includes a .tf file will prompt `atlantis` to wake up and run your terraform plan. Atlantis will post the plan's result to your merge request as a comment within a minute or so.

Review and eventually approve the merge request.

### Apply and Merge
Add the comment `atlantis apply` in the approved merge request. This will prompt atlantis to wake up and run your `terraform apply`.

The apply results will be added to your pull request comments by atlantis.

If the apply is successful, your code will automatically be merged with master, your merge request will be closed, and the state lock will be removed in atlantis.

## Managing Terraform State

The following table shows how state is stored based on your installation selection: 

|State Backed|AWS + Github|AWS + Gitlab|Local + Github|
|:--|:--|:--|:--|
|AWS S3 Bucket|X|X| |
|Local - minio in cluster S3 Bucket| | |X|


### AWS Cloud install - `kubefirst cluster init -cloud aws`


Your terraform state is stored in an s3 bucket named `k1-state-store-xxxxxx`.

The s3 bucket implements versioning, so if your terraform state store ever gets corrupted, you can roll it back to a previous state without too much trouble.

Note that terraform at times needs to store secrets in your state store, and therefore access to this s3 bucket should be restricted to only the administrators who need it.


### Local install - `kubefirst local`
Your terraform state is stored in a local in cluster s3 bucket named `kubefirst-state-store` in minio. 

**Attention:** if you k3d cluster is destroyed, the state will be destroyed also. Local destroy, will remove the state once executed. 


## Tips

### How can I use atlantis to add a new user on my github backed installation?

Go to your new gitops repository in your personal GitHub. Navigate to the `gitops` project and edit the file `terraform/users/admins-github.tf`. In this file, you'll see some blocks that represent admin users - the `kubefirst_bot` user, and a commented-out `admin_one` user.


```
module "admin_one" {
  source            = "./modules/user/github"
  acl_policies      = ["admin"]
  email             = "admin@your-company-io.com"
  first_name        = "Admin"
  github_username   = "admin_one_github_username"
  last_name         = "One"
  username          = "aone"
  user_disabled     = false
  userpass_accessor = data.vault_auth_backend.userpass.accessor
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

Once you've provided them this initial password, they can update their own password throughout the platform by updating their user password entity in vault. Anyone can change their own password, and Admins can reset anyone's password. These rules, just like everything else on Kubefirst, can be configured in your new gitops repository.

![](https://user-images.githubusercontent.com/53096417/204801723-602beff0-12f9-45a9-bb9c-4d85a889d1ce.gif)


### How can I use atlantis to add a new user on my gitlab backed installation?

Log into gitlab using the root credentials that were provided to you in your terminal.

Once logged in, navigate to the `gitops` project and edit the file `terraform/users/admin.tf`. In this file, you'll see some blocks that represent admin users:

```
module "admin_one" {
  source   = "./templates/oidc-user"
  admins_group_id    = gitlab_group.admins.id
  developer_group_id = gitlab_group.developer.id
  username           = "admin1"
  fullname           = "Admin One"
  email              = "admin1@yourcompany.com"
  is_admin           = true
}
```

Edit this code replacing the values for the `module name`, `username`, `fullname`, and `email`. There is also a file for your developers at `terraform/users/developers.tf`. You can duplicate those snippets of code in these files to create as many developers and admins as you need.

Commit this change to a **new branch** and create a merge request. This will kick off the Atlantis workflow. Within a minute or so of submitting the merge request, a comment will appear on the merge request that shows the terraform plan with the changes it will be making to your infrastructure. 

To apply these changes, submit a comment on that Merge Request with the following comment text:
```
atlantis apply
```

Doing so will instruct Atlantis to apply the plan. It will report back with the results of the apply within a minute or so.

NOTE: Atlantis merges your Pull Request automatically once an apply is successfully executed. Don't merge terraform merge requests yourself.

Atlantis will always run plans automatically for you when a merge request is opened that changes files mapped in `atlantis.yaml`

Any new users you have created through this process will have their temporary initial passwords stored in Vault. You can access vault using the information provided to you in the terminal as well, and you will find your users' individual initial passwords in the vault secret store `/secrets/users/<username>`. Once you've provided them this initial password, they can update their password throughout the platform by updating their GitLab user password in their gitlab profile.

![](../../img/kubefirst/getting-started/vault-users.png)

### What else can I use atlantis & terraform for?

For example, you can use your gitops repo to help track the creation of repos:

- [aws+github repo template](https://github.com/kubefirst/gitops-template/blob/main/terraform/github/repos.tf)
- [aws+gitlab repo template](https://github.com/kubefirst/gitops-template/blob/main/terraform/gitlab/kubefirst-repos.tf)
- [local+github repo template](https://github.com/kubefirst/gitops-template/blob/main/localhost/terraform/github/repos.tf)


With terraform using the S3 based state store, you can add any terraform file to the gitops repo on which [atlantis is listeting for](https://github.com/kubefirst/gitops-template/blob/main/atlantis.yaml) and atlantis will try to plan and when approved to apply such plan for you. 
