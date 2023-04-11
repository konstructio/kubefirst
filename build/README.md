# Overview 

This page provides several ways to install and use the kubefirst cli.

# Brew

```bash
brew install kubefirst/tools/kubefirst
kubefirst help
```

# Linux Download

```bash
# AWS Prerequesite (if running kubefirst aws commands)

In order for the CLI to work, We assume you gave your [AWS Credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) files at: `$HOME/.aws`, and [AWS IAM Authenticator](https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html) dependency, that is Helm requirement to authenticate to the EKS cluster.

#Check the release page:
#https://github.com/kubefirst/kubefirst/releases

export KUBEFIRST_VERSION=`curl https://github.com/kubefirst/kubefirst/releases/latest  -Ls -o /dev/null -w %{url_effective} | grep -oE "[^/]+$"`
export BINARY_URL="https://github.com/kubefirst/kubefirst/releases/download/${KUBEFIRST_VERSION}/kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz"
curl -LO $BINARY_URL
tar -xvf kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz -C /usr/local/bin/
chmod +x /usr/local/bin/kubefirst

kubefirst info
```
