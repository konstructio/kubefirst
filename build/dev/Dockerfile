FROM --platform=linux/amd64 golang:1.18

WORKDIR /opt/kubefirst-install
ARG KUBEFIRST_VERSION=1.9.3

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

# install Taskfile
RUN curl -LO https://github.com/go-task/task/releases/download/v3.14.1/task_linux_amd64.tar.gz \
    && mkdir /tmp/taskfile \
    && tar -xvzf task_linux_amd64.tar.gz -C /tmp/taskfile \
    && mv /tmp/taskfile/task /usr/local/bin


# setup user
RUN useradd -ms /bin/bash developer
USER developer
WORKDIR /home/developer/kubefirst

COPY --chown=developer:developer . .

# download dependencies and prepare fresh installation
RUN go mod download
RUN go build -ldflags="-X github.com/kubefirst/kubefirst/configs.K1Version=${KUBEFIRST_VERSION}" .

USER root
RUN  mv ./kubefirst /usr/local/bin/ && chmod 777 /usr/local/bin/kubefirst
USER developer
RUN kubefirst clean
