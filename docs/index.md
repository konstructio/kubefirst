# Choose your adventure

The Kubefirst Platform now supports 3 installation types

## Kubefirst Local (GitHub Only)

This is the fastest way to check otu the kubefirst platform. This installation type will automatically create a local k3d cluster on your laptop, put a gitops repository in your personal GitHub account, and bootstrap the local cluster against that new repository. You will be able to run gitops deployments, build images, publish helm charts, and even run automated infrastructure as code, all without a cloud account or a domain requirement.

## Kubefirston AWS (GitHub or GitLab)

Our AWS cloud platform supports both [GitHub](https://www.github.com) and [GitLab](https://www.gitlab.com) as git providers. 

If you choose **GitLab**, your Git Provider will be self-hosted, meaning Kubefirst will install **GitLab** into your newly created kubernetes cluster. If you choose **GitHub**, the Kubefirst platform will leverage the free **GitHub** system at github.com.

<center>

|                                Kubefirst Local                                       |                                Kubefirst with GitHub                                 |                          Kubefirst with Self Hosted GitLab                           |
|:------------------------------------------------------------------------------------:|:------------------------------------------------------------------------------------:|:------------------------------------------------------------------------------------:|
|  [![Kubefirst](./img/kubefirst/icons/k-ray.png)](./kubefirst/local/install.md)   | [![GitHub](./img/kubefirst/icons/github-200x200.png)](./kubefirst/github/install.md) | [![GitLab](./img/kubefirst/icons/gitlab-200x200.png)](./kubefirst/gitlab/install.md) |
|                     [Team K-Ray!](./kubefirst/local/install.md)                      |                    [Team Octocat!](./kubefirst/github/install.md)                    |                    [Team Tanuki!](./kubefirst/gitlab/install.md)                     |

</center>

If you want to know more about Kubefirst before choosing GitHub or GitLab, check out our [overview](./kubefirst/overview.md).
