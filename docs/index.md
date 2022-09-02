# Welcome

Kubefirst is a fully automated and operational open source platform that includes some the best tools available in the kubernetes space, all working together from a single command. By running `kubefirst cluster create` against your empty aws cloud account, you'll get a gitops cloud management and application delivery ecosystem complete with automated terraform workflows, vault secrets management, gitlab integrations with argo, and an example app that demonstrates how it all pieces together.

![../img/kubefirst/oss-plat-arch.png](../img/kubefirst/oss-plat-arch.png)

---

## Install Overview

- The `kubefirst` cli runs on your localhost and will create an eks cluster that includes gitlab, vault, some argo products, and an example microservice app to demonstrate how everything on the platform works.
- The install takes about 30 minutes to execute. Day-2 operations can't commonly be done within the same hour of cluster provisioning. Kubefirst is solving this on our open source platform. We really hope that's worth a [github star](https://github.com/kubefirst/kubefirst) to you (top right corner).
- Your self-hosted gitlab will come preconfigured with two git repositories `kubefirst/gitops` and `kubefirst/metaphor`.
- All of the infrastructure as code will be in your `gitops` repository in the terraform directory. IAC workflows are fully automated with atlantis by merely opening a merge request against the `gitops` repository.
- All of the applications running in your kubernetes cluster are registered in the `gitops` repository in the root `/registry` directory.
- The `metaphor` repository only needs an update to the main branch to deliver the example application to your new development, staging, and production environments. It will hook into your new vault for secrets, demonstrate automated certs, automated dns, and gitops application delivery. Our ci/cd is powered by argo cd, argo workflows, gitlab, gitlab-runner, and vault.
- The result will be the most comprehensive start to managing a kubernetes-centric cloud entirely on open source that you keep and can adjust as you see fit. It's an exceptional fully functioning starting point, with the most comprehensive scope we've ever seen in open source.
- We'd love to advise your project on next steps - see our available white glove and commercial services.

Note: This infrastructure will run in your AWS cloud and is subject to associated aws fees - it costs about $10/day USD to run. Removal of this infrastructure is also automated with a single `kubefirst destroy` command.

