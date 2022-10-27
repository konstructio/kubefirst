# Choose your adventure

To try kubefirst locally with no costs, check out our new `kubefirst local` command here:
https://kubefirst.io/blog/kubefirst-v110-release-notes

The kubefirst platform now supports both the [GitHub](https://www.github.com) and [GitLab](https://www.gitlab.com) Git providers. 
The Git provider you choose will host the code for you applications, your new gitops repository, and will be fully configured through that gitops repository with some new Terraform that will manage your team and user access to the system.

If you choose **GitLab**, your Git Provider will be self-hosted, meaning Kubefirst will install **GitLab** into your newly created kubernetes
cluster. If you choose **GitHub**, the Kubefirst platform will leverage the free **GitHub** system at github.com.

<center>

|                                Kubefirst with GitHub                                 |                          Kubefirst with Self Hosted GitLab                           |
|:------------------------------------------------------------------------------------:|:------------------------------------------------------------------------------------:|
| [![GitHub](./img/kubefirst/icons/github-200x200.png)](./kubefirst/github/install.md) | [![GitLab](./img/kubefirst/icons/gitlab-200x200.png)](./kubefirst/gitlab/install.md) |
 |                    [Team Octocat!](./kubefirst/github/install.md)                    |                    [Team Tanuki!](./kubefirst/gitlab/install.md)                     |

</center>

If you want to know more about Kubefirst before choosing GitHub or GitLab, check out our [overview](./kubefirst/overview.md).
