# Overview 

Kubefirst cli installation options.

# Homebrew

```bash
brew install kubefirst/tools/kubefirst
```

```bash
kubefirst help
```

# Linux Download

```bash
export KUBEFIRST_VERSION=`curl https://github.com/kubefirst/kubefirst/releases/latest  -Ls -o /dev/null -w %{url_effective} | grep -oE "[^/]+$"`
```

```bash
export BINARY_URL="https://github.com/kubefirst/kubefirst/releases/download/${KUBEFIRST_VERSION}/kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz"
```

```bash
curl -LO $BINARY_URL && \
  tar -xvf kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz -C /usr/local/bin/ && \
  chmod +x /usr/local/bin/kubefirst
```

```bash
kubefirst info
```
