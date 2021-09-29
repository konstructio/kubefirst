#!/usr/bin/env bash

export ARGOCD_AUTH_PASSWORD=N9OlGEfDfqgd0FiK
export ARGOCD_AUTH_USERNAME=admin
export ARGOCD_INSECURE=false
export ARGOCD_SERVER=argocd.starter.kubefirst.com:443
export ARGO_SERVER_URL=argo.starter.kubefirst.com:443
export ATLANTIS_GITLAB_HOSTNAME=gitlab.starter.kubefirst.com
export ATLANTIS_GITLAB_TOKEN=kpoH2AfMDf4SJvW5RAw4
export ATLANTIS_GITLAB_USER=kubefirst
export ATLANTIS_GITLAB_WEBHOOK_SECRET=
export AWS_ACCESS_KEY_ID=AKIAR3B33MDMONOWZLWJ
export AWS_DEFAULT_REGION=us-east-1
export AWS_ROLE_TO_ASSUME=arn:aws:iam::126827061464:role/KubernetesAdmin
export AWS_SECRET_ACCESS_KEY=K9feomEqqpcYYtVdApayYu0tmv1JTJAz4SwyC9D7
export AWS_SESSION_NAME=GitHubAction
export GITLAB_BASE_URL=https://gitlab.starter.kubefirst.com
export GITLAB_TOKEN=kpoH2AfMDf4SJvW5RAw4
export KEYCLOAK_CLIENT_ID=admin-cli
export KEYCLOAK_PASSWORD=Q7yibwtig0
export KEYCLOAK_REALM=master
export KEYCLOAK_URL=https://keycloak.starter.kubefirst.com
export KEYCLOAK_USER=gitlab-bot
export TF_VAR_argo_redirect_uris=["https://argo.starter.kubefirst.com/argo/oauth2/callback"]
export TF_VAR_argocd_auth_password=N9OlGEfDfqgd0FiK
export TF_VAR_argocd_redirect_uris=["https://argocd.starter.kubefirst.com/auth/callback","https://argocd.starter.kubefirst.com/applications"]
export TF_VAR_atlantis_gitlab_token=kpoH2AfMDf4SJvW5RAw4
export TF_VAR_atlantis_gitlab_webhook_secret=
export TF_VAR_aws_access_key_id=AKIAR3B33MDMONOWZLWJ
export TF_VAR_aws_account_id=126827061464
export TF_VAR_aws_region=us-east-1
export TF_VAR_aws_secret_access_key=K9feomEqqpcYYtVdApayYu0tmv1JTJAz4SwyC9D7
export TF_VAR_email_address=devops+starter@kubefirst.com
export TF_VAR_email_domain=kubefirst.com
export TF_VAR_gitlab_bot_root_password=kpoH2AfMDf4SJvW5RAw4
export TF_VAR_gitlab_redirect_uris=["https://gitlab.starter.kubefirst.com"]
export TF_VAR_gitlab_runner_token=Bh2EPJmzThjFXSCjDgtp
export TF_VAR_gitlab_token=kpoH2AfMDf4SJvW5RAw4
export TF_VAR_gitlab_url=gitlab.starter.kubefirst.com
export TF_VAR_hosted_zone_id=/hostedzone/Z0894979RGNIP7HU126B
export TF_VAR_hosted_zone_name=starter.kubefirst.com
export TF_VAR_iam_user_arn=arn:aws:iam::126827061464:user/starter-bot
export TF_VAR_keycloak_admin_password=
export TF_VAR_keycloak_password=Q7yibwtig0
export TF_VAR_keycloak_vault_oidc_client_secret=
export TF_VAR_vault_addr=https://vault.starter.kubefirst.com
export TF_VAR_vault_redirect_uris=["https://vault.starter.kubefirst.com/ui/vault/auth/oidc/oidc/callback","http://localhost:8200/ui/vault/auth/oidc/oidc/callback","http://localhost:8250/oidc/callback","https://vault.starter.kubefirst.com:8250/oidc/callback"]
export TF_VAR_vault_token=s.TxbpJwXvj6j4hNGAEpM1WJUW
export VAULT_ADDR=https://vault.starter.kubefirst.com
export VAULT_TOKEN=s.TxbpJwXvj6j4hNGAEpM1WJUW

###
# usage: ./scripts/nebulous/destroy.sh
###

set -e

source ~/.profile

echo
echo
echo
echo
echo
echo
echo ' __        ___.           _____.__                 __   '
echo '|  | ____ _\_ |__   _____/ ____\__|______  _______/  |_ '
echo '|  |/ /  |  \ __ \_/ __ \   __\|  \_  __ \/  ___/\   __\'
echo '|    <|  |  / \_\ \  ___/|  |  |  ||  | \/\___ \  |  |  '
echo '|__|_ \____/|___  /\___  >__|  |__||__|  /____  > |__|  '
echo '     \/         \/     \/                     \/        '


echo
echo
echo "      https://docs.kubefirst.com"
echo
echo
echo
echo

sleep 12

echo "establishing kubectl config"
K8S_CLUSTER_NAME=kubefirst
aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name $K8S_CLUSTER_NAME
chmod 0600 ~/.kube/config

echo "pulling secrets from secret/atlantis"
export VAULT_TOKEN=$(kubectl -n vault get secret vault-unseal-keys -ojson | jq -r '.data."cluster-keys.json"' | base64 -d | jq -r .root_token)
export VAULT_ADDR="https://vault.${AWS_HOSTED_ZONE_NAME}"
vault login $VAULT_TOKEN
$(echo $(vault kv get -format=json secret/atlantis | jq -r .data.data) | jq -r 'keys[] as $k | "export \($k)=\(.[$k])"')

if [ -z "$SKIP_VAULT_APPLY" ]
then
  echo "forcefully destroying argo, gitlab, and chartmuseum buckets (leaving state store intact)"
  aws s3 rb s3://k1-argo-artifacts-$BUCKET_RAND --force
  aws s3 rb s3://k1-gitlab-backup-$BUCKET_RAND --force
  aws s3 rb s3://k1-chartmuseum-$BUCKET_RAND --force

  echo "##############################################################"
  echo "#"
  echo "# RETAIN SECRETS!! MAY BE NEEDED AFTER VAULT IS DESTROYED:"
  echo "# (you can safely discard after destroy is complete)"
  echo "#"
  echo $(vault kv get -format=json secret/atlantis | jq -r .data.data) | jq -r 'keys[] as $k | "export \($k)=\(.[$k])"'
  echo "##############################################################"
  
  cd /git/gitops/terraform/vault
  echo "destroying vault terraform"
  terraform init
  terraform destroy -target module.bootstrap -auto-approve
  echo "vault terraform destroy complete"
fi


if [ -z "$SKIP_ARGOCD_APPLY" ]
then
  cd /git/gitops/terraform/argocd
  echo "destroying argocd terraform"
  terraform init 
  terraform destroy -target module.argocd_registry -target module.argocd_repos -auto-approve
  echo "argocd terraform destroy complete"

  echo "waiting 240 seconds for app registration destruction"
  sleep 30
  echo "waiting 210 more seconds for app registration destruction"
  sleep 30
  echo "waiting 180 more seconds for app registration destruction"
  sleep 30
  echo "waiting 150 more seconds for app registration destruction"
  sleep 30
  echo "waiting 120 more seconds for app registration destruction"
  sleep 30
  echo "waiting 90 more seconds for app registration destruction"
  sleep 30
  echo "waiting 60 more seconds for app registration destruction"
  sleep 30
  echo "waiting 30 more seconds for app registration destruction"
  sleep 30
fi


if [ -z "$SKIP_GITLAB_APPLY" ]
then
  cd /git/gitops/terraform/gitlab
  echo "destroying gitlab terraform"
  terraform init 
  terraform destroy -auto-approve
  echo "gitlab terraform destroy complete"
fi


if [ -z "$SKIP_BASE_APPLY" ]
then
  cd /git/gitops/terraform/base
  echo "destroying base terraform"
  terraform init 
  terraform destroy -auto-approve
  echo "base terraform destroy complete"
fi


echo "teardown operation complete"

