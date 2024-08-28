FROM --platform=linux/amd64 buildpack-deps:bullseye-scm
# we are using buidlpack-deps:bullseye-scm https://github.com/docker-library/golang/blob/8d0fa6028120904e16fe761f095bd0620b68eab2/1.18/bullseye/Dockerfile

ARG KUBEFIRST_VERSION=1.10.5

RUN apt-get update && \
    apt-get install -y unzip curl jq vim unzip less \
    && rm -rf /var/lib/apt/lists/* 

# Kubernetes client
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.21.3/bin/$(uname -s)/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv kubectl /usr/local/bin/

# AWS cli
RUN curl -LO https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip awscli-exe-linux-x86_64.zip && \
    ./aws/install && \
    rm -r aws && \
    rm awscli-exe-linux-x86_64.zip

# AWS EKS cli
RUN curl -LO https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_linux_amd64.tar.gz && \
    tar -xvzf eksctl_linux_amd64.tar.gz -C /usr/local/bin/ && \
    rm eksctl_linux_amd64.tar.gz

# AWS IAM Authenticator tool
RUN curl -LO https://s3.us-west-2.amazonaws.com/amazon-eks/1.21.2/2021-07-05/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x aws-iam-authenticator && \
    mv aws-iam-authenticator /usr/local/bin/

# change shell from bin/sh to bin/bash
SHELL ["/bin/bash", "-c"]

# Kubefirst cli
RUN curl -LO https://github.com/konstructio/kubefirst/releases/download/$KUBEFIRST_VERSION/kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz && \
    tar -xvzf kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz -C /usr/local/bin/ && \
    chmod +x /usr/local/bin/kubefirst && \
    rm kubefirst_${KUBEFIRST_VERSION:1}_linux_amd64.tar.gz

# setup user
RUN useradd -ms /bin/bash developer
USER developer
WORKDIR /home/developer/kubefirst

RUN kubefirst version
