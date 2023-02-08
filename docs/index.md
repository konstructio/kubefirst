# Kubefirst Platforms

## What is Kubefirst?

Kubefirst is a fully automated and operational open source platform that includes some of the most popular open source tools available in the 
Kubernetes space, all working together from a single command. 

We support local, AWS, and Civo clouds. By running our cli commands against your empty environment, you'll get a GitOps cloud management and application delivery ecosystem complete with automated 
Terraform workflows, Vault secrets management, GitLab or GitHub integrations with Argo, and demo applications 
that demonstrate how it all pieces together.

![](./img/kubefirst/kubefirst-arch.png)

## Installation Types

### Kubefirst Local (GitHub Only)

This is the **fastest** way to check out the kubefirst platform. This installation type will automatically create a local k3d cluster on your laptop, put a gitops repository in your personal GitHub account, and bootstrap the local cluster against that new repository. You will be able to run gitops deployments, build images, publish helm charts, and even run automated infrastructure as code, all without a cloud account or a domain requirement.

### Kubefirst on CIVO Cloud (GitHub Only)

The perfect cloud environment when Kubernetes will be the center of attention. A **simple cloud footprint** with a powerful open source cloud native tool set for identity management, infrastructure management, application delivery, and secrets management.

### Kubefirst on AWS (GitHub or GitLab)

Our AWS cloud platform can accommodate all of the **needs of your enterprise** and supports both [GitHub](https://www.github.com) and [GitLab](https://www.gitlab.com) providers. 

The GitHub option will leverage the free GitHub system at github.com.

The GitLab option will move your git repositories to your newly created kubefirst management cluster.

---

## Choose Your Platform

|                                    |                                    |
|:----------------------------------:|:----------------------------------:|
| **[local k3d with github](./kubefirst/local/install.md)**<br /><br />[![Kubefirst](./img/kubefirst/icons/k-ray.png)](./kubefirst/local/install.md)   | **[civo cloud with github](./kubefirst/github/install.md)**<br /><br /><br /><br />[![Civo](./img/kubefirst/icons/civo.png)](./kubefirst/civo/install.md)  |
| **[aws with github](./kubefirst/github/install.md)**<br /><br />[![GitHub](./img/kubefirst/icons/github-200x200.png)](./kubefirst/github/install.md)   | **[aws with self-hosted gitlab](./kubefirst/gitlab/install.md)**<br /><br />[![GitLab](./img/kubefirst/icons/gitlab-200x200.png)](./kubefirst/gitlab/install.md)   |

To learn more about Kubefirst check out our [overview](./kubefirst/overview.md).
