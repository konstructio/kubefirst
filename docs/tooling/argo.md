# Argo

`argo` is our command line interface to our Argo Workflows

### Releases

Argo keeps their client binaries available on their releases page:   
[https://github.com/argoproj/argo-workflows/releases](https://github.com/argoproj/argo-workflows/releases)

### Installation Instructions
**Warning: the following install is pinned, see [releases page](https://github.com/argoproj/argo-workflows/releases) for a newer version.**
```bash
VERSION=3.1.3; curl -sLO https://github.com/argoproj/argo-workflows/releases/download/v${VERSION}/argo-darwin-amd64.gz
gunzip argo-darwin-amd64.gz
chmod +x argo-darwin-amd64
mv ./argo-darwin-amd64 /usr/local/bin/argo
```
details: [https://github.com/argoproj/argo-workflows/tags](https://github.com/argoproj/argo-workflows/tags)

### Checking Your Argo Version

```bash
argo version --short
```
expected result: `argo: v3.1.3`