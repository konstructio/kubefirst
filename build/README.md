## Runing inside docker

```bash
docker run \
  -it --name kubefirst  \
  --dns="1.0.0.1" --dns="208.67.222.222" --dns="8.8.8.8" \
  -v $(PWD):/opt/kubefirst \
  -v $HOME/.aws:/home/developer/.aws \
   pagottoo/kubefirst_cli:1.8.5
```

## Clone the repository

Clone the repository to have the latest `main` branch content

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

Open a new terminal to connect to the container to run kubefirst

```bash
docker exec -it kubefirst bash
```

