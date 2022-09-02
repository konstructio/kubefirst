# Argo CD

`argocd` is our command line interface to our Argo CD gitops platform

### Releases

ArgoCD keeps their client binaries available on their releases page:   
[https://github.com/argoproj/argo-cd/releases](https://github.com/argoproj/argo-cd/releases)

### Installation Instructions
**Warning: the following install is pinned, see [releases page](https://github.com/argoproj/argo-cd/releases) for a newer version.**
```bash
VERSION=2.0.5; curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/download/v$VERSION/argocd-darwin-amd64
chmod +x /usr/local/bin/argocd
```
details: [https://argoproj.github.io/argo-cd/cli_installation/](https://argoproj.github.io/argo-cd/cli_installation/)

### Checking Your ArgoCD Version

```bash
argocd version​ --short
```
expected result: `argocd: v2.0.5+4c94d88`
