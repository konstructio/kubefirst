# Overview 

This page provide sevral ways to explore kubefirst cli, to allow you to choose the one the better fits your prefered way of work. 


# Requirements to run the CLI

In order for the CLI to work, We assume you gave your [AWS Credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) files at: `$HOME/.aws`. 


# Running CLI in docker container

Based on image: https://hub.docker.com/pagottoo/kubefirst_cli

You can easily run it, without any installation step with:
```bash
docker run \
  -it --name kubefirst  \
  --dns="1.0.0.1" --dns="208.67.222.222" --dns="8.8.8.8" \
  -v $(PWD):/opt/kubefirst \
  -v $HOME/.aws:/home/developer/.aws \
   pagottoo/kubefirst_cli:1.8.5
```

After this step is executed, return to [this step](https://github.com/kubefirst/kubefirst#initialization) to run a `kubefirst info` and other functions.

# Running CLI from a Docker-Compose container

## Clone the repository

Clone the repository to have the latest `main` branch content:

```bash
# via HTTPS
git clone https://github.com/kubefirst/kubefirst.git

# via SSH
git clone git@github.com:kubefirst/kubefirst.git
```

## Start the Container

We run everything in isolation with Docker, for that, start the container with:

```bash
docker-compose up kubefirst
```

## Connect to the Container

Open a new terminal to connect to the container to run kubefirst:

```bash
docker exec -it kubefirst bash
```

After this step is executed, return to [this step](https://github.com/kubefirst/kubefirst#initialization) to run a `kubefirst info` and other functions.
