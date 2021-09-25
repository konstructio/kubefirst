#!/usr/bin/env bash

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
echo "      welcome to the kubefirst/nebulous installation. the install time is about"
echo "      40 mins to provision your new infrastructure. most of this time is waiting"
echo "      on infra to provision and dns to propagate. while you're waiting check out"
echo "      our docs to familiarize yourself with what's ahead."
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
  echo
  echo "############################################################"
  echo "established BUCKET_RAND suffix: ${BUCKET_RAND}"
  echo "set this in your kubefirst.env file for restarts/teardown"
  echo "############################################################"
  echo
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
export TF_VAR_argo_redirect_uris="[\"https://argo.${AWS_HOSTED_ZONE_NAME}/oauth2/callback\"]"
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
  echo
  echo '########################################'
  echo '#'
  echo '#          BASE TERRAFORM'
  echo '#'
  echo '########################################'
  echo

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
  echo
  echo '########################################'
  echo '#'
  echo '#          GITLAB RECONFIG'
  echo '#'
  echo '########################################'
  echo

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
  echo
  echo '########################################'
  echo '#'
  echo '#          GITLAB RECONFIG'
  echo '#'
  echo '########################################'
  echo

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

  echo "updating kubefirst group image"
  export KUBEFIRST_GROUP_ID=$(curl -s --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "https://gitlab.${AWS_HOSTED_ZONE_NAME}/api/v4/groups" | jq -r '.[] | select(.name == "kubefirst").id')
  curl --request PUT --header "PRIVATE-TOKEN: ${GITLAB_TOKEN}" "https://gitlab.${AWS_HOSTED_ZONE_NAME}/api/v4/groups/${KUBEFIRST_GROUP_ID}" --form "avatar=@/images/kubefirst.png"
  echo "group image update complete"
else
  echo "skipping gitlab terraform because SKIP_GITLAB_APPLY is set"
fi

echo
echo '########################################'
echo '#'
echo '#          ARGOCD TERRAFORM'
echo '#'
echo '########################################'
echo

echo "creating argocd in kubefirst cluster"
kubectl create namespace argocd --dry-run -oyaml | kubectl apply -f -
kubectl create secret -n argocd generic aws-creds --from-literal=AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} --from-literal=AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} --dry-run -oyaml | kubectl apply -f -
# kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f - # TODO: kubernetes 1.19 and above
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
echo "argocd created"

echo "sleeping 60 seconds after argocd creation"
sleep 10
echo "sleeping 50 more seconds"
sleep 10
echo "sleeping 40 more seconds"
sleep 10
echo "sleeping 30 more seconds"
sleep 10
echo "sleeping 20 more seconds"
sleep 10
echo "sleeping 10 more seconds"
sleep 10

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



echo "ARGOCD_AUTH_PASSWORD: $ARGOCD_AUTH_PASSWORD"
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
  echo
  echo '########################################'
  echo '#'
  echo '#          VAULT TERRAFORM'
  echo '#'
  echo '########################################'
  echo

  cd /git/gitops/terraform/vault
  echo "applying vault terraform"
  terraform init 
  terraform apply -target module.bootstrap -auto-approve
  # terraform destroy -target module.bootstrap -auto-approve; exit 1 # TODO: hack
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

if [ -z "$SKIP_SSH_STORAGE" ]
then
  echo "writing ssh key to secret/ssh"
  vault login -no-print $VAULT_TOKEN
  vault kv put secret/ssh/terraform_ssh_key terraform_ssh_key_base64="$(cat /git/gitops/terraform/base/terraform-ssh-key | base64)"
  vault kv put secret/ssh/terraform_ssh_key_pub terraform_ssh_key_pub_base64="$(cat /git/gitops/terraform/base/terraform-ssh-key.pub | base64)"
fi

# the following comnmand is a bit fickle as the vault dns propagates, 
# a retry attempts to make this a bit more fault tolerant to that
echo "argocd app sync of gitlab-runner"
for i in 1 2 3 4 5; do argocd app sync gitlab-runner-components && break || echo "sync of gitlab-runner did not complete successfully. this is often due to delays in dns propagation. sleeping for 60s before retry" && sleep 60; done
echo "argocd app sync of chartmuseum"
for i in 1 2 3 4 5; do argocd app sync chartmuseum-components && break || echo "sync of chartmuseum did not complete successfully. this is often due to delays in dns propagation. sleeping for 60s before retry" && sleep 60; done
echo "argocd app sync of keycloak"
for i in 1 2 3 4 5; do argocd app sync keycloak-components && break || echo "sync of keycloak did not complete successfully. this is often due to delays in dns propagation. sleeping for 60s before retry" && sleep 60; done
echo "argocd app sync of atlantis"
for i in 1 2 3 4 5; do argocd app sync atlantis-components && break || echo "sync of atlantis did not complete successfully. this is often due to delays in dns propagation. sleeping for 60s before retry" && sleep 60; done

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

/scripts/nebulous/wait-for-200.sh "https://keycloak.${AWS_HOSTED_ZONE_NAME}/auth/"

#! assumes keycloak has been registered, needed for terraform
export KEYCLOAK_PASSWORD=$(kubectl -n keycloak get secret/keycloak  -ojson | jq -r '.data."admin-password"' | base64 -d)
export KEYCLOAK_USER=gitlab-bot
export KEYCLOAK_CLIENT_ID=admin-cli
export KEYCLOAK_URL=https://keycloak.${AWS_HOSTED_ZONE_NAME}
echo "collect keycloak password $KEYCLOAK_PASSWORD"

# apply terraform
if [ -z "$SKIP_KEYCLOAK_APPLY" ]
then
  echo
  echo '########################################'
  echo '#'
  echo '#          KEYCLOAK TERRAFORM'
  echo '#'
  echo '########################################'
  echo

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

  echo "argocd app sync of argo-components after keycloak secrets exist"
  for i in 1 2 3 4 5 6 7 8; do argocd app sync argo-components && break || echo "sync of argo did not complete successfully. this is often due to delays in dns propagation. sleeping for 60s before retry" && sleep 60; done
  echo "awaiting successful sync of argo-components"
  argocd app wait argo-components
  echo "sync and wait of argo-components and argo is complete"
  
else
  echo "skipping keycloak terraform because SKIP_KEYCLOAK_APPLY is set"
fi

echo "configuring git client"
cd /git/metaphor
git config --global user.name "Administrator"
git config --global user.email "${EMAIL_ADDRESS}"

echo "triggering metaphor delivery pipeline"
cd /git/metaphor
git pull origin main --rebase
git commit --allow-empty -m "kubefirst trigger pipeline"
git push -u origin main 
echo "metaphor delivery pipeline invoked"

echo "triggering gitops pull request to test atlantis workflows"
cd /git/gitops
git pull origin main --rebase
SUFFIX=$RANDOM
git checkout -b "atlantis-test-${SUFFIX}"
echo "" >> /git/gitops/terraform/argocd/main.tf
echo "" >> /git/gitops/terraform/base/main.tf
echo "" >> /git/gitops/terraform/gitlab/kubefirst-repos.tf
echo "" >> /git/gitops/terraform/keycloak/main.tf
echo "" >> /git/gitops/terraform/vault/main.tf
git add .
git commit -m "test the atlantis workflow"
git push -u origin atlantis-test-${SUFFIX} -o merge_request.create
echo "gitops pull request created"


if [ -z "$SKIP_OIDC_PATCHING" ]
then
  echo
  echo '########################################'
  echo '#'
  echo '#            OIDC PATCHING'
  echo '#'
  echo '########################################'
  echo
  
  echo "pulling secrets from secret/admin/oidc-clients/argocd"
  export VAULT_TOKEN=$(kubectl -n vault get secret vault-unseal-keys -ojson | jq -r '.data."cluster-keys.json"' | base64 -d | jq -r .root_token)
  export VAULT_ADDR="https://vault.${AWS_HOSTED_ZONE_NAME}"
  vault login $VAULT_TOKEN
  $(echo $(vault kv get -format=json secret/admin/oidc-clients/argocd | jq -r .data.data) | jq -r 'keys[] as $k | "export \($k)=\(.[$k])"')
  
  echo "adding keycloak configs to argocd configmap"
  kubectl -n argocd patch secret argocd-secret -p "{\"stringData\": {\"oidc.keycloak.clientSecret\": \"${ARGOCD_CLIENT_SECRET}\"}}"
  
  echo "configuring git client"
  cd "/git/gitops"
  git config --global user.name "Administrator"
  git config --global user.email "${EMAIL_ADDRESS}"
  git checkout main
  git pull origin main --rebase
  
  echo "adding oidc config to argocd gitops registry"
  cat << EOF >> /git/gitops/components/argocd/configmap.yaml
  
  url: https://argocd.${AWS_HOSTED_ZONE_NAME}
  oidc.config: |
    name: Keycloak
    issuer: https://keycloak.${AWS_HOSTED_ZONE_NAME}/auth/realms/kubefirst
    clientID: argocd
    clientSecret: \$oidc.keycloak.clientSecret
    requestedScopes: ["openid", "profile", "email", "groups"]

EOF
  
  git add .
  git commit -m "updated oidc config for argocd"
  git push -u origin main
  echo "pushed to gitops origin"
  # argocd app get argocd --hard-refresh


  cd /git/gitops/terraform/vault
  echo "applying vault terraform"
  terraform init 
  terraform apply -auto-approve
  # terraform destroy -auto-approve; exit 1 # TODO: hack
  echo "vault terraform complete"
fi


echo
echo
echo 
echo 
echo
echo
echo
echo "#############################################################################################"
echo "#"
echo "#      !!!! !!! !!! !!! KEEP EVERYTHING PRINTED FROM THIS LINE DOWN !!!! !!! !!! !!!"
echo "#"
echo "#############################################################################################"
sleep 5
echo
echo
echo "|--------------------------------------------------------"
echo "| GitLab"
echo "| https://gitlab.${AWS_HOSTED_ZONE_NAME}"
echo "| username: root"
echo "| password: ${GITLAB_BOT_ROOT_PASSWORD}"
echo "| Repos:"
echo "| https://gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/gitops"
echo "| https://gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/metaphor"
echo "| * keycloak oidc established"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Vault"
echo "| https://vault.${AWS_HOSTED_ZONE_NAME}"
echo "| method: token"
echo "| token: ${VAULT_TOKEN}"
echo "| * keycloak sso enabled"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Argo CD"
echo "| https://argocd.${AWS_HOSTED_ZONE_NAME}"
echo "| username: admin"
echo "| password: ${ARGOCD_AUTH_PASSWORD}"
echo "| * keycloak sso enabled"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Argo Workflows"
echo "| https://argo.${AWS_HOSTED_ZONE_NAME}"
echo "| sso credentials only"
echo "| * keycloak sso enabled"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Keycloak"
echo "| https://keycloak.${AWS_HOSTED_ZONE_NAME}"
echo "| username: gitlab-bot"
echo "| password: ${KEYCLOAK_PASSWORD}"
echo "| * keycloak sso enabled"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Atlantis"
echo "| https://atlantis.${AWS_HOSTED_ZONE_NAME}"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Chart Museum"
echo "| https://chartmuseum.${AWS_HOSTED_ZONE_NAME}"
echo "| see vault for credentials"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Metaphor"
echo "| Development: https://XXX.${AWS_HOSTED_ZONE_NAME}"
echo "| Staging: https://XXX.${AWS_HOSTED_ZONE_NAME}"
echo "| Production: https://XXX.${AWS_HOSTED_ZONE_NAME}"
echo "|--------------------------------------------------------"
sleep 1
echo ""
echo ""
echo "|--------------------------------------------------------"
echo "| Kubernetes backend utilities"
echo "|--------------------------------------------------------"
echo "| Nginx Ingress Controller | ingress-nginx namespace    |"
echo "| Cert Manager             | cert-manager namespace     |"
echo "| Certificate Issuers      | clusterwide                |"
echo "| External Secrets         | external-secrets namespace |"
echo "| GitLab Runner            | gitlab-runner namespace    |"
echo "|--------------------------------------------------------"
sleep 2
echo ""
echo ""
echo ""
echo ""
echo ""
echo "WARNING: Test your connection to Kubernetes, GitLab, and Vault BEFORE CLOSING THIS WINDOW. Connection details follow."
echo "Docs to install the tools mentioned: https://docs.kubefirst.com/tooling/tooling-overview/"
echo ""
echo ""
echo ""
echo ""
echo ""
echo "#####################################################"
echo "# GITLAB"
echo ""
echo "To connect to your GitLab visit: https://gitlab.${AWS_HOSTED_ZONE_NAME}"
echo "username: root"
echo "password: ${GITLAB_BOT_ROOT_PASSWORD}"
echo ""
echo "Once logged into GitLab, visit:"
echo "- the gitops repo: https://gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/gitops"
echo "- the metaphor repo: https://gitlab.${AWS_HOSTED_ZONE_NAME}/kubefirst/metaphor"
echo ""
echo "Please note: your self-hosted GitLab server has Open Registration, which allows"
echo "anyone to sign themselves up. To change this configuration see Sign-up Restrictions"
echo "https://gitlab.${AWS_HOSTED_ZONE_NAME}/admin/application_settings/general"
echo "#####################################################"
echo ""
echo ""
echo "#####################################################"
echo "KUBERNETES"
echo ""
echo "To connect to your kubernetes cluster, install the aws cli and run the following in your terminal:"
echo "export AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID}"
echo "export AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}"
echo "export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION}"
echo "aws eks update-kubeconfig --name kubefirst"
echo ""
echo "To test your new connection, install kubectl and run a few kubectl commands in your terminal:"
echo "kubectl get nodes -owide"
echo "kubectl get namespaces"
echo "#####################################################"
echo ""
echo ""
echo "#####################################################"
echo "VAULT"
echo ""
echo "To login to Vault visit https://vault.${AWS_HOSTED_ZONE_NAME}"
echo "Method: Token"
echo "Token: ${VAULT_TOKEN}"
echo ""
echo "Once you're logged into Vault confirm you can navigate to your infrastructure secrets in Vault at path:"
echo "secret/atlantis"
echo "#####################################################"
echo ""
echo ""
echo "#####################################################"
echo "MISC"
echo ""
echo "Your BUCKET_RAND value is: ${BUCKET_RAND}. You may need this value if you decide to run teardown."
echo "#####################################################"
echo ""
echo ""
echo ""
echo ""
echo "Once you've stored this output and tested the above connections,"
echo "you should continue exploring the kubefirst platform from our docs:"
echo ""
echo "https://docs.kubefirst.com/kubefirst/getting-started/        <--- seriously."    
echo ""
echo ""
echo ""
echo ""
echo "We poured our hearts into this project for 2 years to get you to this message."
echo "We would LOOOOOOVE a github star if you think we earned it."
echo "Top-right corner of this page: https://github.com/kubefirst/nebulous"
echo "Thanks so much crew."
echo ""
echo "- The Kubefirst Team"
echo ""
echo ""
