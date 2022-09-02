# Kubectl

`kubectl` is the command line interface to our kubernetes clusters.

### Releases

Kubernetes keeps their client binaries available on their releases page:   
[https://kubernetes.io/releases/](https://kubernetes.io/releases/)

### Installation Instructions
**Warning: the following install is pinned, see [releases page](https://kubernetes.io/releases/) for a newer version.**
```bash
VERSION=1.21.3; curl -LO "https://dl.k8s.io/release/v${VERSION}/bin/darwin/amd64/kubectl"
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
```
details: [https://kubernetes.io/docs/tasks/tools/](https://kubernetes.io/docs/tasks/tools/)

### Checking Your Kubectl Version

```bash
kubectl version --client --short
```
expected result: `Client Version: v1.21.3`

## Getting EKS Configs

See [https://docs.aws.amazon.com/cli/latest/reference/eks/update-kubeconfig.html](https://docs.aws.amazon.com/cli/latest/reference/eks/update-kubeconfig.html)
To establish or adjust `kubecfg`, your local kubectl config file, with connection details for each of the kubernetes clusters, run the following 3 commands

```bash
AWS_PROFILE=mgmt aws eks update-kubeconfig --name k8s-mgmt --region us-east-2
AWS_PROFILE=preprod aws eks update-kubeconfig --name k8s-preprod --region us-east-2
AWS_PROFILE=production aws eks update-kubeconfig --name k8s-production --region us-east-2
```
