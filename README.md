# nebulous
The Kubefirst Open Source Starter Plan repository

# docker commands

```bash
docker build -t kubefirst/nebulous:0.1-rc4 .

docker push kubefirst/nebulous:0.1-rc4

docker run --env-file=kubefirst.env \
  -v $PWD/terraform:/terraform \
  --entrypoint /scripts/nebulous/init.sh \
  kubefirst/nebulous:0.1-rc4

# DEBUG / interact with tooling in the container
docker run -it --env-file=kubefirst.env \
  -v $PWD/terraform:/terraform \
  --entrypoint /bin/bash \
  kubefirst/nebulous:0.1-rc4
```