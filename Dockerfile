FROM ubuntu:focal

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
  && apt-get --no-install-recommends install -y \
    apt-transport-https \
    apt-utils \
    aufs-tools \
    automake \
    bash \
    bash-completion \
    bsdutils \
    build-essential \
    ca-certificates \
    coreutils \
    curl \
    findutils \
    git \
    gnupg-agent \
    gnupg \
    gnupg2 \ 
    golang-go \
    file \
    grep \
    groff \
    jq \
    less \
    libgtk2.0-0 \
    libgtk-3-0 \
    libgbm-dev \
    libnotify-dev \
    libgconf-2-4 \
    libssl-dev \
    libnss3 \
    libxss1 \
    libasound2 \
    libxtst6 \
    lsb-release \
    make \
    manpages-dev \
    openssh-client \
    openssl \
    postgresql \
    python3 \
    python3-pip \
    sed \
    software-properties-common \
    ssh \
    unzip \
    util-linux \
    vim \
    wget \
    xvfb \
    xauth \
    zip \
 && rm -rf /var/lib/apt/lists/*

RUN add-apt-repository ppa:git-core/ppa -y \
  && apt-get update \
  && apt-get install git -y

RUN pip3 install pyyaml semver --upgrade

# install docker
RUN curl -fsSL https://get.docker.com -o get-docker.sh
RUN /bin/sh get-docker.sh

# install aws cli v2
RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" \
  && unzip awscliv2.zip \
  && ./aws/install

# Installation of NVM, NPM and packages
RUN mkdir /usr/local/nvm
ENV NVM_DIR /usr/local/nvm
ENV NODE_VERSION 14.15.1
ENV NVM_INSTALL_PATH $NVM_DIR/versions/node/v$NODE_VERSION
RUN rm /bin/sh && ln -s /bin/bash /bin/sh
RUN curl --silent -o- https://raw.githubusercontent.com/creationix/nvm/v0.37.2/install.sh | bash
RUN source $NVM_DIR/nvm.sh \
   && nvm install $NODE_VERSION \
   && nvm alias default $NODE_VERSION \
   && nvm use default
ENV NODE_PATH $NVM_INSTALL_PATH/lib/node_modules
ENV PATH $NVM_INSTALL_PATH/bin:$PATH

# install kubectl 
ENV KUBECTL_VERSION v1.17.17 
ADD https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl

# install  hashicorp terraform
ENV TF_VERSION 0.13.5
RUN curl -LO https://releases.hashicorp.com/terraform/${TF_VERSION}/terraform_${TF_VERSION}_linux_amd64.zip \
  && unzip terraform_${TF_VERSION}_linux_amd64.zip \
  && chmod +x terraform \
  && mv terraform /usr/local/bin/terraform \
  && rm -f terraform_${TF_VERSION}_linux_amd64.zip

# install hashicorp vault
ENV VAULT_VERSION 1.6.2
RUN curl -LO https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip \
  && unzip vault_${VAULT_VERSION}_linux_amd64.zip \
  && chmod +x vault \
  && mv vault /usr/local/bin/vault \
  && rm -f vault_${VAULT_VERSION}_linux_amd64.zip

# install helm v3
# DESIRED_VERSION is the helm version to install
ENV DESIRED_VERSION v3.5.0
RUN mkdir -p $HOME/.helm && export HELM_HOME="$HOME/.helm" && curl -L https://git.io/get_helm.sh | /bin/bash
RUN helm plugin install https://github.com/chartmuseum/helm-push.git

# install argocd
RUN ARGOCD_VERSION=$(curl --silent "https://api.github.com/repos/argoproj/argo-cd/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/') \
&& curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/download/$ARGOCD_VERSION/argocd-linux-amd64 \
&& chmod +x /usr/local/bin/argocd \
&& argocd version --short --client

# install argo
RUN ARGO_VERSION="v3.1.11" \
&& curl -sLO https://github.com/argoproj/argo-workflows/releases/download/$ARGO_VERSION/argo-linux-amd64.gz \
&& gunzip argo-linux-amd64.gz \
&& chmod +x argo-linux-amd64 \
&& mv ./argo-linux-amd64 /usr/local/bin/argo \
&& argo version --short

ADD scripts/nebulous /scripts/nebulous
ADD gitops /gitops
ADD metaphor /metaphor
ADD images /images

RUN apt-get update
RUN apt-get install dnsutils -y

CMD [ "/bin/bash" ]
