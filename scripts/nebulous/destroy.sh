#!/usr/bin/env bash

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

echo "forcefully destroying argo, gitlab, and chartmuseum buckets (leaving state store in tact)"
aws s3 rb s3://k1-argo-artifacts-$BUCKET_RAND --force
aws s3 rb s3://k1-gitlab-backup-$BUCKET_RAND --force
aws s3 rb s3://k1-chartmuseum-$BUCKET_RAND --force

if [ -z "$SKIP_VAULT_APPLY" ]
then
  echo "##############################################################"
  echo "# SECRETS, IF NEEDED, AFTER VAULT IS DESTROYED:"
  echo ""
  echo $(vault kv get -format=json secret/atlantis | jq -r .data.data) | jq -r 'keys[] as $k | "export \($k)=\(.[$k])"'
  echo "##############################################################"
  
  cd /git/gitops/terraform/vault
  echo "destroying vault terraform"
  terraform init
  terraform destroy -auto-approve;
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

