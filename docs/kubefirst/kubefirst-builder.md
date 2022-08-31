# Kubefirst Builder

### Introduction to the `kubefirst-builder`

The `kubefirst-builder` is a container image that includes all of the utilities that we need to run our CI layer. It also serves as an example for how to create your own CI images. The `kubefirst-builder` happens to have nodejs, golang, kubectl, helm, and a handful of other utilities. You should feel free to add to them as your automation needs evolve.

The `kubefirst-builder` is the container used by a lot of our gitlab CI jobs, as well as some of our argo workflow templates. To use the `kubefirst-builder` image as the image that your GitLab CI will run upon, simply specify the image in your .gitlab-ci.yaml as shown here:
```
deploy_development:
  variables:
    BUILDER_IMAGE: "${ECR_REGISTRY_BASE_URL}/kubefirst-builder:1.3"
  image: "${BUILDER_IMAGE}"
  ...
```

In the above example, the deploy_development job will leverage the `kubefirst-builder` at tag `1.3`. This tag provides the ability to version your changes to your CI image, adopt the change in the `metaphor` application, test it to make sure the change has the desired effect, and then adopt it in the rest of your applications.

### `kubefirst-builder` Utilities

The `kubefirst-builder's` Dockerfile lays out all the tooling available to our CI. If you need a programming language or bash utility, this is usually where it would be added.

The docker-builder is built from docker:latest, which is an alpine image. Many of the utilities we depend upon are installed using the apk add at the top of this image.
```
FROM docker:latest

RUN apk --no-cache add \
  ansible \
  bash \
  binutils \
  binutils-gold \
  coreutils \
  curl \
  findutils \
  g++ \
  gcc \
  git \
  gnupg \
  go \
  grep \
  jq \
  libc6-compat \
  libgcc \
  libstdc++ \
  linux-headers \
  make \
  musl-dev \
  openssh-client \
  openssl \
  postgresql \
  postgresql-dev \
  python3 \
  python3-dev \
  py3-pip \
  sed \
  unzip \
  util-linux \
  zip \
  && rm -rf /var/cache/apk/*
```

Other utilities like argocd and argo demonstrate how we install utilities that aren't available through the apk package manager.

```
# install argocd
RUN ARGOCD_VERSION=$(curl --silent "https://api.github.com/repos/argoproj/argo-cd/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/') \
&& curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/download/$ARGOCD_VERSION/argocd-linux-amd64 \
&& chmod +x /usr/local/bin/argocd \
&& argocd version --short --client
```

### The `repo-scripts` directory

The `kubefirst-builder` also has a directory called `repo-scripts` which produces the `/scripts` directory in our published images. This approach affords us a dedicates space to add any bash scripting that we want available to all of our CI for common CI/CD tasks.

### The `setup-build-env` script

There is also a dedicated `setup-build-env` script that serves as a layer to run any late-bound components to your CI image. We use this layer to setup our aws and git configuration. If you have a utility that you want pulled onto your image when the job starts, you could add that scripting at this layer and invoke `setup-build-env` at the start of your CI. You'll notice our metaphor example has the following snippet:
```
before_script:
  - source /setup-build-env
```
which runs the setup-build-env bash script at the start of every CI job execution in metaphor.

### Making changes to the `kubefirst-builder`

To add a utility to `kubefirst-builder` and use it in your application's CI, follow this flow:
- add the utility to `kubefirst-builder`
    - if you want it baked into the `kubefirst-builder` image add it to Dockerfile
    - if you instead want it late-bound to the image, add it to the setup-build-env scripting
- tag the `kubefirst-builder` version with a new tag
    - increment the version with a minor version if it's a non-breaking change (eg. 1.3 -> 1.4)
    - increment the version with a major version bump if it's a breaking change (eg. 1.3 -> 2.0)
    - the tag will start a CI pipeline that will build and publish `kubefirst-builder` to your ECR registry
- adopt the new version of your `kubefirst-builder` image in metaphor (just change the tag of the kubefirst image to the new version you tagged in your .gitlab-ci.yml file)
- test that the metaphor app can build and deploy using your new builder without any issue
- once you've confirmed that the new version is working as expected, adopt the new version in your other applications

### Managing builders

The `kubefirst-builer` provides a nice mechanism to ensure that our utilities and languages all conform to a single version across the platform, but there's nothing that requires you to leverage the `kubefirst-builder` specifically. There may be compelling reasons to use a different builder image - for example maybe you want a java-builder that's specifically for managing dependency utilities for a java ecosystem without bloating the `kubefirst-builder` with additional resources.

Any builders can follow the same pattern established by the `kubefirst-builder`. Just copy the Dockerfile and .gitlab-ci.yml to your new builder repo and make the adjustments that are appropriate for your use case. You're also not required to use a custom builder - you can use ubuntu, docker, cypress, or any other publically available OCI image for your CI layer.

### Docker In Docker

One of the utilities included on the `kubefirst-builder` is docker. This gets a little complicated, because your CI is already running on a container in kubernetes. In order to run docker commands on a docker image, we'll need to run docker-in-docker. Running docker-in-docker (or dind for short) requires a few extra steps. You can see this on displat in the publish_container step of metaphor.

```
variables:
  DOCKER_TLS_CERTDIR: "/certs"
  DOCKER_TLS_VERIFY: 1
  DOCKER_CERT_PATH: "${DOCKER_TLS_CERTDIR}/client"
  BUILDER_IMAGE: "${ECR_REGISTRY_BASE_URL}/kubefirst-builder:1.3"

...

publish_container:
  services:
    - docker:19.03.13-dind
  variables:
    DOCKER_DRIVER: overlay
    REGION: "us-east-2"
    AWS_PROFILE: "default"
  image: "${BUILDER_IMAGE}"
  stage: publish-container
  only:
    - master
    - /hotfix/
  script:
    - export DOCKER_HOST=tcp://localhost:2376
    - /scripts/docker/build.sh "${ECR_REGISTRY_BASE_URL}" "${CI_PROJECT_NAME}" "${CI_COMMIT_SHA}"
    - /scripts/aws/ecr-login.sh "${REGION}" "${ECR_REGISTRY_BASE_URL}"
    - /scripts/docker/publish.sh "${ECR_REGISTRY_BASE_URL}" "${CI_PROJECT_NAME}" "${CI_COMMIT_SHA}"
```

Note there are a few `DOCKER_*` variables established for all metaphor jobs, and a `DOCKER_DRIVER` variable set in the individual publish_container job that uses docker. Also note that the service `docker:19.03.13-dind` has been added to the publish_container job. This setup allows the gitlab job to establish a privledged docker sidecar that can connect to the docker daemon that runs on the kubernetes node in a secure way.
