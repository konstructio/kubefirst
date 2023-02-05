# Kubefirst GitOps CI/CD

The days of scripting `kubectl apply` and `helm install` in your operations are over. The kubefirst platform comes with all the tools you need to run a scalable gitops platform powered by argo cd gitops.

![](../img/kubefirst/gitops/gitops-flow.png)

## The Benefits of GitOps

The benefit of defining your desired state in a declarative way in Git is enormous.

- An audit log of every system change including who made and who approved the change is available.
- A secure main branch of the GitOps repo makes for very easy security and approval controls for system changes.
- Easy to understand rollbacks (just revert the Git commit that caused the problem and you'll be on the last version).
- The declarative state in Git significantly reduces the complexity of disaster recovery.
- Creating new environments is as straight-forward as duplicating directory structures.
- Having everything declared in the GitOps repo means everyone can agree to the source of truth for every version of every app.

## Pipelines

![](../img/kubefirst/gitops/gitops-cicd.png)

Our CI pipelines originate in your privately hosted git repo trigger mechanism(Github Actions or Gitlab CI) with a workflow being submitted to Argo Workflows. The workflows will publish your container with a pre-release Helm chart.

The pipelines then continue with promotion of the release through the preprod environments.

Once the release is ready, the release chart is published and delivered to production, and the application chart's version will be prepared for the next release.

### GitHub Integration with Argo Workflows

![](../img/kubefirst/github/github-argo-workflow.png)

The integration that we've established between GitHub and Argo Workflows will surface your powerful Argo Workflows engine directly in GitHub Actions, so that your developers have all of the workflow execution information available, directly associated with their commit in their application repository.

### GitLab Integration with Argo Workflows

![](../img/kubefirst/gitops/gitlab-workflows-integration.png)

The integration that we've established between GitLab and Argo Workflows will surface your powerful Argo Workflows engine directly in GitLab's native CI, so that your developers have all of the workflow execution information available directly associated with their commit in their application repository.

![](../img/kubefirst/gitlab/gitlab-argo-workflow.png)

## GitOps Resources

Our registry includes GitOps patterns for numerous source types including:

- Helm Charts
- Wrapper Charts
- Kubernetes Manifest Files

The metaphor example application includes an example of how easy it is to set different configuration overrides for your different environments.

To see what it takes to make the `development` instance of `metaphor` different than the others, visit the `gitops` repo and navigate to `/components/development/metaphor/values.yaml`

Any Helm value that should deviate from the default chart can be set here so your environment overrides are always concise and clear.
