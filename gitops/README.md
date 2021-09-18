![](logo.png)

# gitops

- argocd desired state
- terraform prepped for gitops

## terraform

The terraform in this repository can be found in the `/terraform/` directory. 

All of our terraform is automated with atlantis. To see the terraform entry points and how they are triggered, see [atlantis.yaml](./atlantis.yaml).

## argocd

The argocd configurations in this repo can be found in the [registry directory](./registry). Your applications will also go here in the development, staging, and production folder. The metaphor app can be found there to serve as an example to follow.
