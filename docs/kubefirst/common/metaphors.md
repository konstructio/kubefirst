# Metaphor

**Metaphors** are Kubefirst demo applications to demonstrate how a regular application can be integrated into the 
Kubefirst platform following best practices. The demo applications consists of **Metaphor frontend**, 
**Metaphor Go API**, and **Metaphor NodeJS API**.

### Best Practices

The **Metaphors** applications provides a demonstration space for application best practices in a tangible way that's 
easy to apply to other projects. When engineers discover good patterns and wish have those patterns adopted in other 
projects, add that new idea in the most straightforward way possible to the Metaphors applications as well. By doing so 
our engineering team can fully engage with the best application approaches.

### CI/CD

The **Metaphors** applications has a complete CI/CD process with automated builds, container creation, container 
publishing, linting, tests, deployment to `preprod` and `production`, release management, version management, and 
hotfixes. It serves as an excellent proving ground for changes to the CI/CD layer.

When a new version of our CI is needed, it's best to adopt that new version of the CI in the **Metaphors** application
first. Run through the adjustments to your automation and test it through all of your environments and kubernetes 
clusters without impacting the applications that your engineers and users depend on.

### Kubernetes Representations

The **Metaphors** applications are multi-instance load balanced applications. It's deployed to the `development` and 
`staging` namespaces in the `preprod` cluster, and the `production` namespace in the `production` cluster.

![](../../img/kubefirst/metaphor/metaphor-kubernetes-manifests.png)

The kubernetes manifests produced by the **Metaphors** applications CI include a working example of a kubernetes 
deployment with downstream replicaset and pods, a service account with a security context used, a service to make the 
application available to the cluster, and an ingress to make the service available outside the cluster.

### Ingress Integrations

The ingress manifest demonstrates integration with our automated approach to DNS management, load balancer management, 
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

### Some cool automated things to note:

- the value specified in `spec.rules.host` will automatically create a route53 cname that's bound to the ingress elastic load balancer
- the `cert-manager.io/cluster-issuer` annotation will prompt `cert-manager` to automatically create a certificate for your application, and will store that cert in the `secretName` specified
- nginx will automatically route traffic to the **Metaphors** applications service based on the path-based/host-based routing specified in `spec.rules`

### Environment Configs and Secrets

The **Metaphors** applications also includes a working example of how to leverage a multi-environment secrets management 
paradigm powered by **Vault** and `external-secrets`.

There's also a configmap implementation to demonstrate how to leverage non-sensitive configuration values.

### Datadog Integrations

The **Metaphors** applications are set up to provide cloud and container observability and monitoring best practices 
with **Datadog**. It demonstrates using **Datadog** for **Metaphors** application logs, container statistics, application 
metrics, application performance monitoring, dashboard, and alerting.

### Secrets Management

The **Metaphors** applications leverages hashicorp **Vault** for secrets management. **Vault** runs in the `mgmt` cluster 
and metaphor runs in `preprod` and `production`, so it serves as an example for secrets management. To read more see our 
[vault page](../gitlab/vault.md)
