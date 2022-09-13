# What is Kubefirst

Kubefirst is a fully automated and operational open source platform that includes some the best tools available in the 
Kubernetes space, all working together from a single command. By running `kubefirst cluster create` against your empty 
aws cloud account, you'll get a GitOps cloud management and application delivery ecosystem complete with automated 
terraform workflows, vault secrets management, **GitLab** or **GitHub** integrations with Argo, and demo applications 
that demonstrates how it all pieces together.

## Install overview

[//]: # (todo: update wording)
- The `kubefirst` cli runs on your localhost and will create an **AWS EKS cluster** that includes **GitLab** or **GitHub**, **Vault**, **ArgoCD** and **Argo Workflow**, and example applications with the **Metaphors** application to demonstrate how everything on the platform works.
- The install takes about 30 minutes to execute. Day-2 operations can't commonly be done within the same hour of cluster provisioning. Kubefirst is solving this on our open source platform. We really hope that's worth a [GitHub Kubefirst repository](https://github.com/kubefirst/kubefirst) star to you (top right corner).
- Your self-hosted GitLab or cloud GitHub will come preconfigured with two git repositories `kubefirst/gitops` and `kubefirst/metaphor-frontend`, `kubefirst/metaphor-go`, `kubefirst/metaphor`.
- All of the infrastructure as code will be in your GitOps repository in the Terraform directory. IAC workflows are fully automated with atlantis by merely opening a merge request against the `gitops` repository.
- All of the applications running in your kubernetes cluster are registered in the GitOps repository in the root `/registry` directory.
- The Metaphors repositories only needs an update to the main branch to deliver the example application to your new development, staging, and production environments. It will hook into your new Vault for secrets, demonstrate automated certs, automated DNS, and GitOps application delivery. Our CI/CD is powered by ArgoCD, Argo Workflows, GitLab or GitHub, and Vault.
- The result will be the most comprehensive start to managing a kubernetes-centric cloud entirely on open source that you keep and can adjust as you see fit. It's an exceptional fully functioning starting point, with the most comprehensive scope we've ever seen in open source.
- We'd love to advise your project on next steps - see our available white glove and commercial services.

_Note: This infrastructure will run in your AWS cloud and is subject to associated aws fees - it costs about $10/day USD 
to run. Removal of this infrastructure is also automated with a single kubefirst destroy command._

## Console UI (WIP)

[//]: # (todo: add description / add screen shot)

## Destroying

Kubefirst also makes it easy to destroy a previsioned cluster. By calling `kubefirst destroy` command, all previsioned
resources are deleted. This is a process that takes some minutes to be finished, since all created resources needs to 
be properly destroyed.

One step that takes some minutes to conclude is the EKS cluster deletion, and the reason is because Kubefirst destroy
every resource that was created during the installation, including VPC, load balancer, sub networks and everything else
that was created like ArgoCD, Argo Workflow, demo applications, and GitLab self-hosted.
