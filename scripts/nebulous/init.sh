#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/init.sh
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
echo "      hi, welcome to the kubefirst/nebulous installation. the install time is about"
echo "      15 - 20 min to provision your new infrastructure. while you're waiting"
echo "      we recommend checking out our docs to familiarize yourself with what's ahead"
echo
echo "      https://docs.kubefirst.com"
echo
echo
echo
echo

sleep 12

echo "executing source-profile.sh"
source /scripts/nebulous/source-profile.sh 

# conditional configuration setup
if [ ! -f /gitops/terraform/base/terraform-ssh-key ]
then
    echo "creating ssh key pair"
    ssh-keygen -o -t rsa -b 4096 -C "${EMAIL_ADDRESS}" -f $HOME/.ssh/id_rsa -q -N "" > /dev/null
    echo "copying ssh keys to /gitops/terraform/base/terraform-ssh-key*"
    cp ~/.ssh/id_rsa /gitops/terraform/base/terraform-ssh-key
    cp ~/.ssh/id_rsa.pub /gitops/terraform/base/terraform-ssh-key.pub
    sleep 2
else
    echo "reusing existing ssh key pair"
fi

if [ -z "$BUCKET_RAND" ]; then
  BUCKET_RAND=$(openssl rand -hex 3)
  echo "established new bucket suffix ${BUCKET_RAND}"
else
  echo "BUCKET_RAND seeded with specified value $BUCKET_RAND, skipping state bucket creation"
  SKIP_STATE_BUCKET_CREATION="true"
fi


export TF_STATE_BUCKET=k1-state-store-$BUCKET_RAND
export ARGO_ARTIFACT_BUCKET=k1-argo-artifacts-$BUCKET_RAND
export GITLAB_BACKUP_BUCKET=k1-gitlab-backup-$BUCKET_RAND
export CHARTMUSEUM_BUCKET=k1-chartmuseum-$BUCKET_RAND


if [ -z "$SKIP_STATE_BUCKET_CREATION" ]
then
    echo "creating bucket $TF_STATE_BUCKET"
    # TODO: --versioning-configuration Status=Enabled
    if [[ "$AWS_DEFAULT_REGION" == "us-east-1" ]]; then
      aws s3api create-bucket --bucket $TF_STATE_BUCKET --region $AWS_DEFAULT_REGION | jq -r .Location | cut -d/ -f2
    else
      aws s3api create-bucket --bucket $TF_STATE_BUCKET --region $AWS_DEFAULT_REGION --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION | jq -r .Location | cut -d/ -f3 | cut -d. -f1
    fi
    echo "enabling bucket versioning"
    aws s3api put-bucket-versioning --bucket $TF_STATE_BUCKET --region $AWS_DEFAULT_REGION --versioning-configuration Status=Enabled
else
    echo "reusing bucket name ${TF_STATE_BUCKET}"
fi


# setup environment variables
export AWS_HOSTED_ZONE_ID=$(aws route53 list-hosted-zones-by-name | jq --arg name "${AWS_HOSTED_ZONE_NAME}." -r '.HostedZones | .[] | select(.Name=="\($name)") | .Id')
export EMAIL_DOMAIN=$(echo $EMAIL_ADDRESS |  cut -d"@" -f2)
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity | jq -r .Account)
export IAM_USER_ARN=$(aws sts get-caller-identity | jq -r .Arn)
export GITLAB_URL_PREFIX=gitlab
export GITLAB_URL="${GITLAB_URL_PREFIX}.${AWS_HOSTED_ZONE_NAME}"
export GITLAB_ROOT_USER=root
export GITLAB_BASE_URL=https://gitlab.${AWS_HOSTED_ZONE_NAME}
export K8S_CLUSTER_NAME=kubefirst

#* terraform separation: all these values should come from pre-determined env's
export TF_VAR_aws_account_id=$AWS_ACCOUNT_ID
export TF_VAR_aws_region=$AWS_DEFAULT_REGION
export TF_VAR_hosted_zone_name=$AWS_HOSTED_ZONE_NAME
export TF_VAR_hosted_zone_id=$AWS_HOSTED_ZONE_ID
export TF_VAR_gitlab_url=$GITLAB_URL
export TF_VAR_email_domain=$EMAIL_DOMAIN
export TF_VAR_iam_user_arn=$IAM_USER_ARN
export TF_VAR_gitlab_bot_root_password=$GITLAB_BOT_ROOT_PASSWORD
export TF_VAR_aws_access_key_id=$AWS_ACCESS_KEY_ID
export TF_VAR_aws_secret_access_key=$AWS_SECRET_ACCESS_KEY
export TF_VAR_email_address=$EMAIL_ADDRESS
export TF_VAR_vault_redirect_uris="[\"https://vault.${AWS_HOSTED_ZONE_NAME}/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8200/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8250/oidc/callback\",\"https://vault.${AWS_HOSTED_ZONE_NAME}:8250/oidc/callback\"]"
export TF_VAR_argo_redirect_uris="[\"https://argo.${AWS_HOSTED_ZONE_NAME}/argo/oauth2/callback\"]"
export TF_VAR_argocd_redirect_uris="[\"https://argocd.${AWS_HOSTED_ZONE_NAME}/auth/callback\",\"https://argocd.${AWS_HOSTED_ZONE_NAME}/applications\"]"
export TF_VAR_gitlab_redirect_uris="[\"https://gitlab.${AWS_HOSTED_ZONE_NAME}\"]"


export TF_VAR_keycloak_password=""
export TF_VAR_atlantis_gitlab_webhook_secret=$ATLANTIS_GITLAB_WEBHOOK_SECRET #TODO: created empty - wire it in
export TF_VAR_keycloak_admin_password=""
export TF_VAR_keycloak_vault_oidc_client_secret=$TF_VAR_keycloak_vault_oidc_client_secret


# check for liveness of the hosted zone
if [ -z "$SKIP_HZ_CHECK" ]; then
  HZ_LIVENESS_FAIL_COUNT=0
  HZ_IS_LIVE=0
  HZ_LIVENESS_URL=livenesstest.$AWS_HOSTED_ZONE_NAME
  HZ_LIVENESS_JSON="{\"Comment\":\"CREATE sanity check record \",\"Changes\":[{\"Action\":\"CREATE\",\"ResourceRecordSet\":{\"Name\":\"$HZ_LIVENESS_URL\",\"Type\":\"A\",\"TTL\":300,\"ResourceRecords\":[{\"Value\":\"4.4.4.4\"}]}}]}"
  echo "Creating $HZ_LIVENESS_URL record for sanity check"
  HZ_RECORD_STATUS=$(aws route53 change-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --change-batch "$HZ_LIVENESS_JSON" | jq -r .ChangeInfo.Status)

  while [[ $HZ_RECORD_STATUS == 'PENDING' && $HZ_LIVENESS_FAIL_COUNT -lt 8 && $HZ_IS_LIVE -eq 0 ]];
  do
    echo "checking hosted zone configuration with validation of livenesstest.$AWS_HOSTED_ZONE_NAME"  
    HZ_LOOKUP_RESULT=$(nslookup "$HZ_LIVENESS_URL" 8.8.8.8 | awk -F':' '/^Address: / { matched = 1 } matched { print $2}' | xargs)
    if [[ "$HZ_LOOKUP_RESULT" ]]; then
      HZ_IS_LIVE=1
      echo "Sanity check passed"
    else
      HZ_LIVENESS_FAIL_COUNT=$((HZ_LIVENESS_FAIL_COUNT+1))
      echo "Sanity check url, $HZ_LIVENESS_URL, is not ready yet. Sleeping for 30 seconds"
      sleep 30
    fi
  done

  echo "Deleting $HZ_LIVENESS_URL record"
  aws route53 change-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --change-batch "$( echo "${HZ_LIVENESS_JSON//CREATE/DELETE}" )" > /dev/null

  if [[ $HZ_IS_LIVE -eq 0 ]]; then
    echo "Error creating an A record in the provided hosted zone! we can't go on, check your zone, credentials, region, etc and try again"
    exit 1
  fi
fi

if [ -z "$SKIP_DETOKENIZATION" ]; then
  # detokenize
  export LC_CTYPE=C; 
  export LANG=C;
  echo "copying constructed gitops repo content into /git/gitops directory"
  mkdir -p /git
  cd /git
  cp -a /gitops/. /git/gitops/
  cp -a /metaphor/. /git/metaphor/
  cd /git/

  # NOTE: this section represents values that need not be secrets and can be directly hardcoded in the 
  # clients' gitops repos. DO NOT handle secrets in this fashion
  echo "replacing TF_STATE_BUCKET token with value ${TF_STATE_BUCKET} (1 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<TF_STATE_BUCKET>|${TF_STATE_BUCKET}|g"
  echo "replacing ARGO_ARTIFACT_BUCKET token with value ${ARGO_ARTIFACT_BUCKET} (2 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<ARGO_ARTIFACT_BUCKET>|${ARGO_ARTIFACT_BUCKET}|g"
  echo "replacing GITLAB_BACKUP_BUCKET token with value ${GITLAB_BACKUP_BUCKET} (3 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<GITLAB_BACKUP_BUCKET>|${GITLAB_BACKUP_BUCKET}|g"
  echo "replacing CHARTMUSEUM_BUCKET token with value ${CHARTMUSEUM_BUCKET} (4 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<CHARTMUSEUM_BUCKET>|${CHARTMUSEUM_BUCKET}|g"
  echo "replacing AWS_ACCESS_KEY_ID token with value ${AWS_ACCESS_KEY_ID} (5 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<AWS_ACCESS_KEY_ID>|${AWS_ACCESS_KEY_ID}|g"
  echo "replacing AWS_HOSTED_ZONE_ID token with value ${AWS_HOSTED_ZONE_ID} (6 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<AWS_HOSTED_ZONE_ID>|${AWS_HOSTED_ZONE_ID}|g"
  echo "replacing AWS_HOSTED_ZONE_NAME token with value ${AWS_HOSTED_ZONE_NAME} (7 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<AWS_HOSTED_ZONE_NAME>|${AWS_HOSTED_ZONE_NAME}|g"
  echo "replacing AWS_DEFAULT_REGION token with value ${AWS_DEFAULT_REGION} (8 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<AWS_DEFAULT_REGION>|${AWS_DEFAULT_REGION}|g"
  echo "replacing EMAIL_ADDRESS token with value ${EMAIL_ADDRESS} (9 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<EMAIL_ADDRESS>|${EMAIL_ADDRESS}|g"
  echo "replacing AWS_ACCOUNT_ID token with value ${AWS_ACCOUNT_ID} (10 of 10)"
  find . \( -type d -name .git -prune \) -o -type f -print0 | xargs -0 sed -i "s|<AWS_ACCOUNT_ID>|${AWS_ACCOUNT_ID}|g"
fi

# apply base terraform
cd /git/gitops/terraform/base
if [ -z "$SKIP_BASE_APPLY" ]
then
  echo "applying bootstrap terraform"
  terraform init 
  terraform apply -auto-approve
  # terraform destroy -auto-approve; exit 1; # hack

  KMS_KEY_ID=$(terraform output -json | jq -r '.vault_unseal_kms_key.value')
  echo "KMS_KEY_ID collected: $KMS_KEY_ID"
  echo "bootstrap terraform complete"
  echo "replacing KMS_KEY_ID token with value ${KMS_KEY_ID}"
  
  cd /git/gitops
  find . -type f -not -path '*/cypress/*' -exec sed -i "s|<KMS_KEY_ID>|${KMS_KEY_ID}|g" {} + 

else
  echo "skipping bootstrap terraform because SKIP_BASE_APPLY is set"
fi

echo "getting ~/kube/config for eks access"
aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name $K8S_CLUSTER_NAME
cat ~/.kube/config
chmod 0600 ~/.kube/config





#! gitlab
if [ -z "$SKIP_GITLAB_RECONFIG" ]
then
  # reconfigure gitlab server
  echo
  echo "testing gitlab for connectivity"
  echo
  /scripts/nebulous/wait-for-200.sh "https://${GITLAB_URL}/help"
  echo
  echo "gitlab is ready, executing cypress"
  echo

  export CYPRESS_BASE_URL="https://${GITLAB_URL}"
  export CYPRESS_gitlab_bot_username_before=$GITLAB_ROOT_USER
  export CYPRESS_gitlab_bot_password=$GITLAB_BOT_ROOT_PASSWORD
  cd /git/gitops/terraform/cypress
  npm ci

  $(npm bin)/cypress run
  
  
  echo
  echo "    IMPORTANT:"
  echo "      THIS IS THE ROOT PASSWORD FOR YOUR GITLAB INSTANCE"
  echo "      DO NOT LOSE THIS VALUE"
  echo
  echo "      username: root"
  echo "      password: ${CYPRESS_gitlab_bot_password}"
  echo
  echo "      GitLab URL: https://${GITLAB_URL}/kubefirst"
  echo
  echo
  echo
  echo "    hydrating your GitLab server's kubefirst group with CI/CD"
  echo "      variables, check it out -> https://$GITLAB_URL/groups/kubefirst/-/settings/ci_cd"
  echo
  echo
  sleep 18
fi


export RUNNER_REGISTRATION_TOKEN=$(cat /git/gitops/terraform/.gitlab-runner-registration-token)
export GITLAB_TOKEN=$(cat /git/gitops/terraform/.gitlab-bot-access-token)
export TF_VAR_gitlab_token=$GITLAB_TOKEN
export TF_VAR_atlantis_gitlab_token=$GITLAB_TOKEN

# apply terraform
if [ -z "$SKIP_GITLAB_APPLY" ]
then
  
  cd /git/gitops/terraform/gitlab
  echo "applying gitlab terraform"
  terraform init 
  terraform apply -auto-approve
  # terraform destroy -auto-approve; exit 1 # TODO: hack
  echo "gitlab terraform complete"
  
  cd /git/gitops
  
  echo "configuring git client"
  git config --global user.name "Administrator"
  git config --global user.email "${EMAIL_ADDRESS}"

  echo "initing gitops repo, committing, and pushing to the new gitlab origin"
  git init --initial-branch=main
  git remote add origin https://root:$GITLAB_TOKEN@gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/gitops.git > /dev/null
  git add .
  git commit -m "initial kubefirst commit"
  git push -u origin main 
  echo "gitops repo established"

  cd /git/metaphor

  echo "configuring git client"
  git config --global user.name "Administrator"
  git config --global user.email "${EMAIL_ADDRESS}"
  
  echo "initing metaphor repo, committing, and pushing to the new gitlab origin"
  git init --initial-branch=main
  git remote add origin https://root:$GITLAB_TOKEN@gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/metaphor.git > /dev/null
  git add .
  git commit -m "initial kubefirst commit"
  git push -u origin main 
  echo "metaphor repo established"

else
  echo "skipping gitlab terraform because SKIP_GITLAB_APPLY is set"
fi



echo "creating argocd in kubefirst cluster"
kubectl create namespace argocd --dry-run -oyaml | kubectl apply -f -
kubectl create secret -n argocd generic aws-creds --from-literal=AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} --from-literal=AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} --dry-run -oyaml | kubectl apply -f -
# kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f - # TODO: kubernetes 1.19 and above
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
echo "argocd created"

echo "sleeping 20 seconds after argocd creation"
sleep 20

echo "connecting to argocd in background process"
kubectl -n argocd port-forward svc/argocd-server -n argocd 8080:443 &
echo "connection to argocd established"

echo "collecting argocd connection details"
export ARGOCD_AUTH_PASSWORD=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
export TF_VAR_argocd_auth_password=${ARGOCD_AUTH_PASSWORD}
export ARGOCD_AUTH_USERNAME=admin
export ARGOCD_INSECURE=true
export ARGOCD_SERVER=localhost:8080

cd /git/gitops/terraform/argocd
echo "applying argocd terraform"
terraform init 
terraform apply -target module.argocd_repos -auto-approve
terraform apply -target module.argocd_registry -auto-approve
# terraform destroy -target module.argocd_registry -target module.argocd_repos -auto-approve; exit 1 # TODO: hack
echo "argocd terraform complete"



echo "password: $ARGOCD_AUTH_PASSWORD"
argocd login localhost:8080 --insecure --username admin --password "${ARGOCD_AUTH_PASSWORD}"
echo "sleeping 120 seconds before checking vault status"
sleep 30
echo "sleeping 90 more seconds before checking vault status"
sleep 30
echo "sleeping 60 more seconds before checking vault status"
sleep 30
echo "sleeping 30 more seconds before checking vault status"
sleep 30
argocd app wait vault

export VAULT_TOKEN=$(kubectl -n vault get secret vault-unseal-keys -ojson | jq -r '.data."cluster-keys.json"' | base64 -d | jq -r .root_token)
export VAULT_ADDR="https://vault.${AWS_HOSTED_ZONE_NAME}"
export TF_VAR_vault_addr=$VAULT_ADDR
export TF_VAR_vault_token=$VAULT_TOKEN
export TF_VAR_gitlab_runner_token=$(cat /git/gitops/terraform/.gitlab-runner-registration-token)


/scripts/nebulous/wait-for-200.sh "https://vault.${AWS_HOSTED_ZONE_NAME}/ui/vault/auth?with=token"

# apply terraform
if [ -z "$SKIP_VAULT_APPLY" ]
then
  cd /git/gitops/terraform/vault
  echo "applying vault terraform"
  terraform init 
  terraform apply -auto-approve
  # terraform destroy -auto-approve; exit 1 # TODO: hack
  echo "vault terraform complete"

  echo "waiting 180 seconds after vault terraform apply"
  sleep 30
  echo "waiting 150 more seconds after vault terraform apply"
  sleep 30
  echo "waiting 120 more seconds after vault terraform apply"
  sleep 30
  echo "waiting 90 more seconds after vault terraform apply"
  sleep 30
  echo "waiting 60 more seconds after vault terraform apply"
  sleep 30
  echo "waiting 30 more seconds after vault terraform apply"
  sleep 30
  
else
  echo "skipping vault terraform because SKIP_VAULT_APPLY is set"
fi

# the following comnmand is a bit fickle as the vault dns propagates, 
# a retry attempts to make this a bit more fault tolerant to that
echo "argocd app sync of gitlab-runner"
for i in 1 2 3 4 5; do argocd app sync gitlab-runner-components && break || echo "sync of gitlab-runner failed, sleeping for 60s before retry" sleep 60; done
echo "argocd app sync of chartmuseum"
for i in 1 2 3 4 5; do argocd app sync chartmuseum-components && break || echo "sync of chartmuseum failed, sleeping for 60s before retry" sleep 60; done
echo "argocd app sync of keycloak"
for i in 1 2 3 4 5; do argocd app sync keycloak-components && break || echo "sync of keycloak failed, sleeping for 60s before retry" sleep 60; done
echo "argocd app sync of atlantis"
for i in 1 2 3 4 5; do argocd app sync atlantis-components && break || echo "sync of atlantis failed, sleeping for 60s before retry" sleep 60; done
echo "argocd app sync of argo"
for i in 1 2 3 4 5; do argocd app sync argo && break || echo "sync of argo failed, sleeping for 60s before retry" sleep 60; done

echo "awaiting successful sync of gitlab-runner"
argocd app wait gitlab-runner-components
argocd app wait gitlab-runner

echo "awaiting successful sync of chartmuseum"
argocd app wait chartmuseum-components
argocd app wait chartmuseum

echo "awaiting successful sync of keycloak"
argocd app wait keycloak-components
argocd app wait keycloak

echo "awaiting successful sync of atlantis"
argocd app wait atlantis-components
argocd app wait atlantis

echo "awaiting successful sync of argo"
argocd app wait argo

/scripts/nebulous/wait-for-200.sh "https://keycloak.${AWS_HOSTED_ZONE_NAME}/auth/"

#! assumes keycloak has been registered, needed for terraform
export KEYCLOAK_PASSWORD=$(kubectl -n keycloak get secret/keycloak  -ojson | jq -r '.data."admin-password"' | base64 -d)
export KEYCLOAK_USER=gitlab-bot
export KEYCLOAK_CLIENT_ID=admin-cli
export KEYCLOAK_URL=https://keycloak.${AWS_HOSTED_ZONE_NAME}

# apply terraform
if [ -z "$SKIP_KEYCLOAK_APPLY" ]
then
  cd /git/gitops/terraform/keycloak
  echo "applying keycloak terraform"
  terraform init 
  terraform apply -auto-approve
  echo "keycloak terraform complete"

  echo "updating vault with keycloak password"
  cd /git/gitops/terraform/vault
  echo "reapplying vault terraform to sync secrets"
  export TF_VAR_keycloak_password=$KEYCLOAK_PASSWORD
  echo "TF_VAR_keycloak_password is $TF_VAR_keycloak_password"
  terraform init
  terraform apply -auto-approve
  echo "vault terraform complete"

  echo "kicking over atlantis pod to pickup latest secrets"
  kubectl -n atlantis delete pod atlantis-0
  echo "atlantis pod has been recycled"   
else
  echo "skipping keycloak terraform because SKIP_KEYCLOAK_APPLY is set"
fi








echo
echo
echo 
echo 
echo
echo
echo "    congratulations, you've made it through the nebulous provisioning process."
echo "    we spent 26 months working all night and weekend for free to get you to this message."
echo
echo "    we would LOOOOOOVE a github star if you made it this far and think we earned it."
echo "    the star is in the top-right corner of this page: https://github.com/kubefirst/nebulous"
echo
echo
echo "      1. visit your new GitLab instance at"
echo "           https://$GITLAB_URL/kubefirst"
echo "      2. sign in with:"
echo "           username: root"
echo "           password: $GITLAB_BOT_ROOT_PASSWORD"
echo "      3. commit to the main branch of metaphor and checkout your pipelines"
echo "         https://$GITLAB_URL/kubefirst/metaphor/-/pipelines"
echo "           app url: metaphor-development.$AWS_HOSTED_ZONE_NAME"
echo
echo
echo
echo
echo
echo
echo
echo
echo
echo
