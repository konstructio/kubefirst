FROM --platform=linux/amd64 golang:1.18

WORKDIR /opt/kubefirst

RUN apt-get update && \
    apt-get install -y unzip


# enable terminal vi mode
RUN set -o vi

# Kubernetes client
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.21.3/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv kubectl /usr/local/bin/

# AWS cli
RUN curl -LO https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip && \
    unzip awscli-exe-linux-x86_64.zip && \
    ./aws/install

# AWS EKS cli
RUN curl -LO https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_linux_amd64.tar.gz && \
    tar -xvzf eksctl_linux_amd64.tar.gz -C /usr/local/bin/

# AWS IAM Authenticator tool
RUN curl -LO https://s3.us-west-2.amazonaws.com/amazon-eks/1.21.2/2021-07-05/bin/linux/amd64/aws-iam-authenticator && \
    chmod +x aws-iam-authenticator && \
    mv aws-iam-authenticator /usr/local/bin/

# setup user
RUN useradd -ms /bin/bash developer
USER developer
WORKDIR /home/developer/kubefirst
