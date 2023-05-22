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

## Manual Install

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

## Install with ASDF ([asdf-kubefirst](https://github.com/Claywd/asdf-kubefirst))
    It's best to run latest but if you already have asdf setup and prefer to use it, there is a kubefirst plugin available to get kubefirst installed and running quickly with asdf.

    Installation instructions for asdf are here. Confirm its installed with `asdf version`
    The [asdf-kubefirst plugin is here](https://github.com/Claywd/asdf-kubefirst)

    Once you have asdf installed, just run the following commands kubefirst will be up and running.

    ```
    asdf plugin-add kubefirst https://github.com/Claywd/asdf-kubefirst.git;
    asdf install kubefirst latest;
    asdf global kubefirst;
    kubefirst version; 
    ```


