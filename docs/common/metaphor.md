# Metaphor

**Metaphor** is a suite of demo microservice applications to demonstrate how an application can be integrated into the 
Kubefirst platform following best practices. The demo applications consists of a **Metaphor frontend**, 
**Metaphor Go API**, and **Metaphor NodeJS API**.




The following table shows what will be installed based with your selection: 

|Application|AWS + Github|AWS + Gitlab|Local + Github|
|:--|:--|:--|:--|
|Metaphor|X|X| |
|Metaphor Go|X|X| |
|Metaphor Frontend|X|X|X|



## Best Practices

The **Metaphors** applications provide a demonstration space for application best practices in a tangible way that's 
easy to apply to other projects. When engineers discover good patterns to use in other 
projects, they can add that new idea in the most straightforward way possible to the Metaphor applications as well. By doing so 
our engineering team can fully engage with the best application approaches.

## CI/CD 

The **Metaphors** applications come with complete CI/CD processes including automated builds, container Helm chart creation, container 
and Helm chart publishing, linting, tests, GitOps deployments to `development`, `staging`, and `production` namespaces, 
release management, version management, and hotfixes. It serves as an excellent proving ground for changes to the CI/CD layer.

When a new version of our CI is needed, it's best to adopt that new version of the CI in a **Metaphor** application
first. Run through the adjustments to your automation and test it through all of your environments and kubernetes 
clusters without impacting the applications that your engineers and users depend on.

## Kubernetes Representations

The **Metaphors** applications are multi-instance load balanced applications. It's deployed to the `development`, 
`staging`, and `production` namespaces in your `kubefirst` cluster.

![](../img/kubefirst/metaphor/metaphor-kubernetes-manifests.png)

The Kubernetes manifests produced by the **Metaphors** applications CI include a working example of a Kubernetes 
deployment with downstream ReplicaSet and pods, a service account with a security context used, a service to make the 
application available to the cluster, and an Ingress to make the service available outside the cluster.

## Ingress Integrations

The Ingress manifest demonstrates integration with our automated approach to DNS management, load balancer management, 
and TLS/SSL certificate creation and renewal.

``` yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: nginx
    # Change the next line to "letsencrypt-staging" while testing adjustments, change to "letsencrypt-prod" after confirming LE certificate was issued
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  name: metaphor
  labels:
    run: metaphor
spec:
  rules:
    - host: metaphor-development.your-company.io
      http:
        paths:
          - backend:
              serviceName: metaphor
              servicePort: 3000
            path: /
  tls:
    - secretName: metaphor-tls
      hosts:
        - metaphor-development.your-company.io
```

## Some Cool Automated Things to Note:

For an AWS Cloud selection(`kubefirst create --cloud aws`):

- the value specified in `spec.rules.host` will automatically create a route53 CNAME that is bound to the Ingress elastic load balancer.
- The `cert-manager.io/cluster-issuer` annotation will prompt `cert-manager` to automatically create a certificate for your application and will store that cert in the `secretName` specified.
- NGINX will automatically route traffic to the **Metaphors** applications service based on the path-based/host-based routing specified in `spec.rules`.

For a local selection(`kubefirst local`):

- A local dns route will be produced with SSL certificates generated for you
- This certificates can be added to your machine truststore to allow a more smooth experience

## Environment Configs and Secrets

The **Metaphors** applications also include a working example of how to leverage a multi-environment secrets management
paradigm powered by **Vault** and `external-secrets`.

There is also a ConfigMap implementation to demonstrate how to leverage non-sensitive configuration values.

## Datadog Integrations

The **Metaphors** applications are set up to provide cloud and container observability and monitoring best practices 
with **Datadog**. It demonstrates using **Datadog** for **Metaphors** application logs, container statistics, application 
metrics, application performance monitoring, dashboard, and alerting.

## Secrets Management

The **Metaphors** applications leverages hashicorp **Vault** for secrets management. **Vault** runs in the `mgmt` cluster 
and metaphor runs in `preprod` and `production`, so it serves as an example for secrets management. To read more see our 
[Vault documentation](../vault/).

## How its CI/CD is defined

These are the key files/folders to be replciated in case, you want to use **Metaphor** to your aplication:

```bash 
.argo
.github
chart/Metaphor
build
.gitlab-ci.yaml
```

- **Concept 1:** If you are using github(local or cloud), it will be trigger based at `.github/workflows/`; Or if you are using gitlab, installation, it will be trigger based at `.gitlab-ci.yaml`. The idea is that these are used for simply triggering an **argo workflows**.

- **Concept 2:** By using **argo workflows** to drive your CI jobs you can re-use some of the **CWFT** we provide and also create your own [**CWFTs**](../tooling/argo/cwft-overview.html) to help build your toolset, the ideia here is to have more generic automations that are not bound to a given git provider tool. 

- **Concept 3:** Use our [**CWFTs**](../tooling/argo/cwft-overview.html) as the basis to build your library of automations by adding new ones that fit your application needs. 

- **Concept 4:** Application is build from a Dockerfile that is defined on the `build` folder. 


## Metaphors and Helm 

We provide a sample application that is packed with helm, you don't need to use helm. if you want to use it, we show how to handle charts update flow based on helm charts and gitops. 

The files you be interested are: 

```bash 
chart/Metaphor
```

There is a [CWFT meant to bump a chart](../tooling/argo/cwft-helm.html#helm-increment-chart-patch) version and update chart museum. This automation is to guide how to leverage the tooling already embeded on kubefirst to serve applications internally. 



## Wrapping up 

Here is described how metaphor gives you a demo of most of the tooling added to your cluster once the installation is finished. It is added in a way that self-unfold once the cluster is ready. 

Want to learn more, check:

- Gitops
- [CWFTs](../../tooling/argo/cwft-overview/)
- [Vault](../../common/vault/)


## Tips

### Metaphor and Local - Special Attention

If you want to use it as base of your application, and bring a new application to a local installation. Be aware, as we use user accounts for local, you need to add a github runner deployment for that new application repo. 

Reference: [runnerdeployment.yaml](https://github.com/kubefirst/gitops-template/blob/main/localhost/components/github-runner/runnerdeployment.yaml)

At your gitops repo go to `components/github-runner/runnerdeployment.yaml` and clone this file, then update the property `spec.template.spec.repository` to point to `your-user/your-repo`. This will deploy a new set of runners to observe that repo for you, allowing CI triggers to be executed. 

```yaml 
...
spec:
  replicas: 1
  template:
    spec:  
      repository: <your-user>/<your-repo>
...
```



### Can I remove Metaphor on my install? 

yes, how to do it:  

- If you are using `kubefirst create cluster` just pass the flag `--skip-metaphor-services` that will prevent the metaphors applications to be installed at your cluster and repos will not be created. 

- If you are using `kubefirst local` just pass the flag `--skip-metaphor` that will prevent the metaphors applications to be installed at your cluster and repos will not be created. 

### Can I add gates to prevent metaphor to move between development to production?

yes, the idea of our current approach of self-unfold to all enviroments it is to allow you to test the tires of of your clusterwith minimal need of clicks on the ui, but yes you can create and add a logic on the deployment artifacts to hold until a giving situation is satisfied. 

You want to be aware of this artifacts at your gitops repo, where the `metaphor` and your applications probably will be added to be deployed on this giving enviroments. 
- components/development
- components/staging
- components/production

### Where metaphor comes from? What repos will be created on my account/org?

If you are using a cloud(`kubefirst cluster create`) selection you have 3 demo applications:

- [metaphor-frontend](https://github.com/kubefirst/metaphor-frontend-template)
- [metaphor](https://github.com/kubefirst/metaphor-template)
- [metaphor-go](https://github.com/kubefirst/metaphor-go-template)

If you are using a local(`kubefirst local`) selection you have 1 demo application:

- [metaphor-frontend](https://github.com/kubefirst/metaphor-frontend-template)
