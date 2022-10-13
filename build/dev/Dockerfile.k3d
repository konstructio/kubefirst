FROM golang:1.18
ARG HOME_PATH=/home/developer
RUN \
    apt-get update && apt-get install -y \
    curl jq vim unzip less \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    passwd  \ 
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /etc/apt/keyrings   && \
    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
    $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    &&  apt-get update \
    && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*



RUN \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.21.3/bin/linux/amd64/kubectl \
    && chmod +x ./kubectl \
    && mv ./kubectl /usr/local/bin/kubectl


RUN \
    curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp \
    &&  mv /tmp/eksctl /usr/local/bin

RUN \
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"   \
    && unzip awscliv2.zip \
    && ./aws/install \
    && aws --version

RUN \
    curl -o aws-iam-authenticator https://s3.us-west-2.amazonaws.com/amazon-eks/1.21.2/2021-07-05/bin/linux/amd64/aws-iam-authenticator \
    && chmod +x ./aws-iam-authenticator \
    &&  mv ./aws-iam-authenticator /usr/local/bin


RUN  go install github.com/spf13/cobra-cli@latest
RUN  go install honnef.co/go/tools/cmd/staticcheck@latest
RUN \
    apt-get update && apt-get install -y \
    curl jq vim unzip less \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir -p ${HOME_PATH}
#RUN useradd -ms /bin/bash developer --home ${HOME_PATH}
RUN echo ${HOME_PATH}
#RUN usermod -m -d ${HOME_PATH} root
#RUN usermod -aG docker developer
RUN chmod 777 -R /go
# USER developer

WORKDIR /app
