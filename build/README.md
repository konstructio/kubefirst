# Overview

kubefirst CLI installation options.

- [macOS with Homebrew](#macos-with-homebrew)
- [macOS or Linux with asdf](#macos-or-linux-with-asdf)
- [Linux Manual Installation](#linux-manual-installation)

If you ran into any issues, please let us know in the #helping-hands channel on our [Slack community](https://kubefirst.io/slack), or by [creating an issue](https://github.com/kubefirst/kubefirst/issues/new?assignees=&labels=bug&projects=&template=bugs.yml).

## macOS with Homebrew

```bash
brew install kubefirst/tools/kubefirst
```

```bash
kubefirst help
```

## macOS or Linux with asdf

> Not maintained by kubefirst

It's best to run the latest version of kubefirst but if you already have asdf setup and prefer to use it, there is a [kubefirst plugin](](https://github.com/Claywd/asdf-kubefirst)) available to get kubefirst installed and running quickly with [asdf](https://asdf-vm.com/). Installation instructions for asdf are [here](https://asdf-vm.com/guide/getting-started.html).

Once you have asdf installed, just run the following commands to install the kubefirst plugin:

```shell
asdf plugin-add kubefirst https://github.com/Claywd/asdf-kubefirst.git
asdf install kubefirst latest
asdf global kubefirst
kubefirst version
```

## Linux Manual Installation

You can download the latest build for your architecture from the [releases page](https://github.com/kubefirst/kubefirst/releases). Alternatively, you can follow these instructions.

First, you need to get the latest version of kubefirst by running this command which will set an environment variable we'll use later:

```shell
export KUBEFIRST_VERSION=`curl https://github.com/kubefirst/kubefirst/releases/latest  -Ls -o /dev/null -w %{url_effective} | grep -oE "[^/]+$"`
```

Secondly, we need to define the architecture of your computer.

```shell
export ARCH=`/usr/bin/arch | sed 's/aarch64/arm64/'`
```

Lastly, we build the URL for downloading the right binary for your computer architecture.

```shell
export BINARY_URL="https://github.com/kubefirst/kubefirst/releases/download/${KUBEFIRST_VERSION}/kubefirst_${KUBEFIRST_VERSION:1}_linux_${ARCH}.tar.gz"
```

Now we can download the file, extract it, and ensure it's executable. You may need to use `sudo` for the `tar` or `chmod`` command.

```shell
curl -L $BINARY_URL -o kubefirst.tar.gz && \
tar --overwrite -xvf kubefirst.tar.gz -C /usr/local/bin/ kubefirst && \
chmod +x /usr/local/bin/kubefirst
```

Now you can run `kubefirst`.

```shell
kubefirst version
```
