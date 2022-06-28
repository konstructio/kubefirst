# Flare

- [Flare](#flare)
  - [Start](#start)
    - [Start Environment variables](#start-environment-variables)
    - [Start Actions](#start-actions)
    - [Start Confirmation](#start-confirmation)
  - [Destroy](#destroy)
    - [Destroy Actions](#destroy-actions)
      - [Notes:](#notes)

## Start

### Start Environment variables

In order to start Kubefirst, the required environment variables are:

| Variable         | example            |
|------------------|--------------------|
| AWS_PROFILE      | started            |
| AWS_REGION       | us-east-1          |
| HOSTED_ZONE_NAME | kubefast.com       |
| ADMIN_EMAIL      | john@kubefirst.com |

### Start Actions

```bash
touch ~/.flare
mkdir -p ~/.kubefirst
cd ~/git/kubefirst/gitlab/flare # change to your dir if different
go build -o bin/flare main.go
./bin/flare nebulous init --admin-email $ADMIN_EMAIL --cloud aws --hosted-zone-name $HOSTED_ZONE_NAME --region $AWS_REGION
./bin/flare nebulous create
```

### Start Confirmation

```bash
aws eks update-kubeconfig --name kubefirst
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
kubectl -n argocd port-forward svc/argocd-server 8080:80
```

## Destroy

To destroy remote then local.

These environment variables are expected:

| Variable         | example                                                                                       |
|------------------|-----------------------------------------------------------------------------------------------|
| AWS_PROFILE      | started                                                                                       |
| AWS_REGION       | us-east-1                                                                                     |
| AWS_ACCOUNT_ID   | 126827061464                                                                                  |
| HOSTED_ZONE_NAME | kubefast.com                                                                                  |
| GITLAB_TOKEN     | "xxxxx1-xx1x-x1xx-1" # replace with value from ~/.flare (only needed if you got to gitlab tf) |


### Destroy Actions
```bash
./bin/flare nebulous destroy
rm -rf ~/.kubefirst
rm ~/.flare
```

#### Notes:

added gitlab.yaml to registry
pushing local to soft origin
