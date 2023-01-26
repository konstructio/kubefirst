The `metaphor-frontend` repo is a simple sample microservice with source code, build, and delivery automation that we use to demonstrate parts of the platform. We also find it to be a valuable way to test CI changes without impacting real apps on your platform.

If you visit your `/.github/workflows/main.yaml` in the `metaphor-frontend` repository, you'll see that it's just sending some workflows to argo workflows in your local k3d cluster.

The example delivery pipeline will:

- Publish the metaphor container to your private github.
- add the metaphor image to a release candidate helm chart and publish it to chartmuseum
- set the metaphor with the desired Helm chart version in the GitOps repo for development and staging
- the release stage of the pipeline will republish the chart, this time without the release candidate notation making it an officially released version and prepare the metaphor application chart for the next release version
- the officially released chart will be set as the desired Helm chart for production.

To watch this pipeline occur, make any change to the `main` branch of of the `metaphor-frontend`. If you're not feeling creative, you can just add a newline to the `README.md`. Once a file in `main` is changed, navigate to metaphor-frontend's CI/CD in the github `Actions` tab to see the workflows get submitted to Argo workflows.

![metaphor-readme-update](../../../img/kubefirst/local/methapor-readme-update.png)

You can visit the metaphor-frontend development, staging, and production apps in your browser to see the versions change as you complete resources and ArgoCD syncs the apps. The metaphor-frontend URLs can be found in your gitops and metaphor-frontend project `README.md` files. 

![metaphor-frontend-development](../../../img/kubefirst/local/methapor-frontend-development.png)

