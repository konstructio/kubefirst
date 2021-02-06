# nebulous
The Kubefirst Open Source Starter Plan repository

# docker commands

```bash
docker build -t kubefirst/nebulous:0.1-rc7 .



docker run --env-file=kubefirst.env \
  -v $PWD/terraform:/terraform \
  --entrypoint /scripts/nebulous/init.sh \
  kubefirst/nebulous:0.1-rc7

docker push kubefirst/nebulous:0.1-rc7

# DEBUG / interact with tooling in the container
docker run -it --env-file=kubefirst.env \
  -v $PWD/terraform:/terraform \
  --entrypoint /bin/bash \
  kubefirst/nebulous:0.1-rc7
```

kubefirst-demo-7466e897ea841b0cce3432bff4a2c8

# todo us-east-1 still doesnt work for some reason.
