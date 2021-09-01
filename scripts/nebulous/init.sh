#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/init.sh
###

set -e

source ~/.profile

#* need to tokenize terraform/base/main.tf
# todo need to move `key` to terraform/base/tfstate.tf
# terraform {
#   backend "s3" {
#     bucket  = "@S3_BUCKET_NAME@"
#     key    = "terraform/tfstate.tf"
#     region  = "@AWS_DEFAULT_REGION@"
#     encrypt = true
#   }
# }

#* need to tokenize /scripts/nebulous/helm/*
# gitlabUrl: @GITLAB_URL@
# runnerRegistrationToken: @RUNNER_REGISTRATION_TOKEN@
# domainFilters:
  # - @HOSTED_ZONE_NAME@


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
echo "      hi, welcome to the Kubefirst Open Source Starter Installation. the install time is about"
echo "      15 - 20 min to provision your new aws infrastructure. while you're waiting"
echo "      we recommend checking out our docs to familiarize yourself with what's ahead"
echo
echo "      https://docs.kubefirst.com/starter"
echo
echo
echo
echo

sleep 12 

echo "executing source-profile.sh"
source /scripts/nebulous/source-profile.sh 

# conditional configuration setup
if [ ! -f $HOME/.ssh/id_rsa ]
then
    echo "creating ssh key pair"
    ssh-keygen -o -t rsa -b 4096 -C "${EMAIL_ADDRESS}" -f $HOME/.ssh/id_rsa -q -N "" > /dev/null
    echo "copying ssh keys to terraform/base/terraform-ssh-key*"
    cp ~/.ssh/id_rsa /terraform/base/terraform-ssh-key
    cp ~/.ssh/id_rsa.pub /terraform/base/terraform-ssh-key.pub
    sleep 2
else
    echo "reusing existing ssh key pair"
fi

BUCKET_RAND=$(openssl rand -hex 15)
export ARGO_ARTIFACT_BUCKET=k1-argo-artifacts-$BUCKET_RAND
export GITLAB_BACKUP_BUCKET=k1-gitlab-backup-$BUCKET_RAND
export CHARTMUSEUM_BUCKET=k1-chartmuseum-$BUCKET_RAND

if [ -z "$TF_STATE_BUCKET_NAME" ]
then
    export TF_STATE_BUCKET_NAME=k1-state-store-$BUCKET_RAND
    echo "creating bucket $TF_STATE_BUCKET_NAME"
    if [[ "$AWS_DEFAULT_REGION" == "us-east-1" ]]; then
      aws s3api create-bucket --bucket $BUCKET_NAME --region $AWS_DEFAULT_REGION --versioning-configuration Status=Enabled | jq -r .Location | cut -d/ -f2
    else
      aws s3api create-bucket --bucket $BUCKET_NAME --region $AWS_DEFAULT_REGION --versioning-configuration Status=Enabled --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION | jq -r .Location | cut -d/ -f3 | cut -d. -f1
    fi
else
    echo "reusing bucket name ${TF_STATE_BUCKET_NAME}"
fi


# setup environment variables
HOSTED_ZONE_NAME=$(aws route53 get-hosted-zone --id "${AWS_HOSTED_ZONE_ID}" | jq -r .HostedZone.Name)
HOSTED_ZONE_NAME=${HOSTED_ZONE_NAME%?}
EMAIL_DOMAIN=$(echo $EMAIL_ADDRESS |  cut -d"@" -f2)
AWS_ACCOUNT_ID=$(aws sts get-caller-identity | jq -r .Account)
IAM_USER_ARN=$(aws sts get-caller-identity | jq -r .Arn)
GITLAB_URL_PREFIX=gitlab
GITLAB_URL="${GITLAB_URL_PREFIX}.${HOSTED_ZONE_NAME}"
GITLAB_ROOT_USER=root

#* terraform separation: all these values should come from pre-determined env's
export TF_VAR_aws_account_id=$AWS_ACCOUNT_ID
export TF_VAR_aws_account_name=starter
export TF_VAR_aws_region=$AWS_DEFAULT_REGION
export TF_VAR_hosted_zone_name=$HOSTED_ZONE_NAME
export TF_VAR_hosted_zone_id=$AWS_HOSTED_ZONE_ID
export TF_VAR_gitlab_url=$GITLAB_URL
export TF_VAR_email_domain=$EMAIL_DOMAIN
export TF_VAR_iam_user_arn=$IAM_USER_ARN
export TF_VAR_gitlab_bot_root_password=$GITLAB_BOT_ROOT_PASSWORD
export TF_VAR_aws_access_key_id=$AWS_ACCESS_KEY_ID
export TF_VAR_aws_secret_access_key=$AWS_SECRET_ACCESS_KEY
export TF_VAR_email_address=$EMAIL_ADDRESS
export TF_VAR_vault_redirect_uris='["https://vault.<AWS_HOSTED_ZONE_NAME>/ui/vault/auth/oidc/oidc/callback","http://localhost:8200/ui/vault/auth/oidc/oidc/callback","http://localhost:8250/oidc/callback","https://vault.<AWS_HOSTED_ZONE_NAME>:8250/oidc/callback"]'
export TF_VAR_argo_redirect_uris='["https://argo.<AWS_HOSTED_ZONE_NAME>/argo/oauth2/callback"]'
export TF_VAR_argocd_redirect_uris='["https://argocd.<AWS_HOSTED_ZONE_NAME>/auth/callback","https://argocd.<AWS_HOSTED_ZONE_NAME>/applications"]'
export TF_VAR_gitlab_redirect_uris='["https://gitlab.<AWS_HOSTED_ZONE_NAME>"]'

# deal with these:
ARGOCD_AUTH_PASSWORD
ATLANTIS_GITLAB_TOKEN
ATLANTIS_GITLAB_WEBHOOK_SECRET
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
GITLAB_TOKEN
KEYCLOAK_PASSWORD
TF_VAR_keycloak_admin_password
TF_VAR_keycloak_vault_oidc_client_secret
VAULT_TOKEN

# check for liveness of the hosted zone
if [ -z "$SKIP_HZ_CHECK" ]
then
  HZ_LIVENESS_FAIL_COUNT=0
  HZ_IS_LIVE=0
  HZ_LIVENESS_URL=livenesstest.$HOSTED_ZONE_NAME
  HZ_LIVENESS_JSON="{\"Comment\":\"CREATE sanity check record \",\"Changes\":[{\"Action\":\"CREATE\",\"ResourceRecordSet\":{\"Name\":\"$HZ_LIVENESS_URL\",\"Type\":\"A\",\"TTL\":300,\"ResourceRecords\":[{\"Value\":\"4.4.4.4\"}]}}]}"
  echo "Creating $HZ_LIVENESS_URL record for sanity check"
  HZ_RECORD_STATUS=$(aws route53 change-resource-record-sets --hosted-zone-id $AWS_HOSTED_ZONE_ID --change-batch "$HZ_LIVENESS_JSON" | jq -r .ChangeInfo.Status)

  while [[ $HZ_RECORD_STATUS == 'PENDING' && $HZ_LIVENESS_FAIL_COUNT -lt 8 && $HZ_IS_LIVE -eq 0 ]];
  do
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


# detokenize terraform - #! need to include every entrypoint gitlab, keycloak, vault, base
sed -i "s|@S3_BUCKET_NAME@|${S3_BUCKET_NAME}|g" "/terraform/base/main.tf"
sed -i "s|@AWS_DEFAULT_REGION@|${AWS_DEFAULT_REGION}|g" "/terraform/base/main.tf"



# detokenize
cd /gitops/
find ./ -type f -exec sed -i -e "s/<TF_STATE_BUCKET>/${TF_STATE_BUCKET}/g" {} \;
find ./ -type f -exec sed -i -e "s/<ARGO_ARTIFACT_BUCKET>/${ARGO_ARTIFACT_BUCKET}/g" {} \;
find ./ -type f -exec sed -i -e "s/<GITLAB_BACKUP_BUCKET>/${GITLAB_BACKUP_BUCKET}/g" {} \;
find ./ -type f -exec sed -i -e "s/<CHARTMUSEUM_BUCKET>/${CHARTMUSEUM_BUCKET}/g" {} \;
find ./ -type f -exec sed -i -e "s/<AWS_ACCESS_KEY_ID>/${AWS_ACCESS_KEY_ID}/g" {} \;
find ./ -type f -exec sed -i -e "s/<AWS_SECRET_ACCESS_KEY>/${AWS_SECRET_ACCESS_KEY}/g" {} \;
find ./ -type f -exec sed -i -e "s/<AWS_HOSTED_ZONE_ID>/${AWS_HOSTED_ZONE_ID}/g" {} \;
find ./ -type f -exec sed -i -e "s/<AWS_HOSTED_ZONE_NAME>/${AWS_HOSTED_ZONE_NAME}/g" {} \;
find ./ -type f -exec sed -i -e "s/<AWS_DEFAULT_REGION>/${AWS_DEFAULT_REGION}/g" {} \;
find ./ -type f -exec sed -i -e "s/<EMAIL_ADDRESS>/${EMAIL_ADDRESS}/g" {} \;



# apply terraform
cd /gitops/terraform/base
if [ -z "$SKIP_TERRAFORM_APPLY" ]
then
  echo "applying bootstrap terraform"
  terraform init 
  terraform destroy -auto-approve #!* todo need to --> apply
  echo "bootstrap terraform complete"
else
  echo "skipping bootstrap terraform because SKIP_TERRAFORM_APPLY is set"
fi

echo "getting ~/kube/config for eks access"
K8S_CLUSTER_NAME=$(terraform output -json | jq -r .cluster_name.value)
aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name $K8S_CLUSTER_NAME
chmod 0600 ~/.kube/config


#! TODO: something has to happen here to argo create the registry in gitops
# steps needed:
# - kubectl apply the argocd installation
# - register the gitops repo with argocd
# - register the registry directory 
# - needs to be automated through 0 touch 

# todo need to add the configmap content for atlantis to use a volume mount and have access to our 
# kubectl -n atlantis create configmap kubeconfig --from-file=config=kubeconfig_kubefirst
# kubernetes clusters - could use secret in vault or configmap as our mount point
# need detokenized gitops directory content for consumption


# --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

#! everything beyond this point needs to be in a workflow or kubernetes job 

# --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
# grab the vault unseal cluster keys, decode to json, parse the root_token
export VAULT_TOKEN=$(kubectl -n vault get secret vault-unseal-keys -ojson | jq -r '.data."cluster-keys.json"' | base64 -d | jq -r .root_token)
export VAULT_ADDR="https://vault.${HOSTED_ZONE_NAME}"
export TF_VAR_vault_addr=$VAULT_ADDR
export TF_VAR_vault_token=$VAULT_TOKEN
export TF_VAR_gitlab_runner_token=$(cat /terraform/.gitlab-runner-registration-token) #! verify path
# TODO: add secrets back in - update: do we need this now that we prioritize external secrets?
# kubectl create namespace kubefirst
# kubectl create secret -n kubefirst generic kubefirst-secrets \
#   --from-literal=AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
#   --from-literal=AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
#   --from-literal=AWS_HOSTED_ZONE_ID=${AWS_HOSTED_ZONE_ID} \
#   --from-literal=AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION} \
#   --from-literal=EMAIL_ADDRESS=${EMAIL_ADDRESS} \
#   --from-literal=GITLAB_BOT_ROOT_PASSWORD=${GITLAB_BOT_ROOT_PASSWORD} \
#   --from-literal=VAULT_TOKEN=${VAULT_TOKEN} \
#   --from-literal=VAULT_ADDR=${VAULT_ADDR}



# apply terraform
if [ -z "$SKIP_VAULT_APPLY" ]
then
  cd /gitops/terraform/vault
  echo "applying vault terraform"
  terraform init 
  terraform destroy -auto-approve #!* todo need to --> apply
  echo "vault terraform complete"
else
  echo "skipping vault terraform because SKIP_VAULT_APPLY is set"
fi


#! assumes keycloak has been registered, needed for terraform
export KEYCLOAK_PASSWORD=$(kubectl -n keycloak get secret/keycloak  -ojson | jq -r '.data."admin-password"' | base64 -d)
export KEYCLOAK_USER=gitlab-bot
export KEYCLOAK_CLIENT_ID=admin-cli
export KEYCLOAK_URL=https://keycloak.<AWS_HOSTED_ZONE_NAME>

# apply terraform
if [ -z "$SKIP_KEYCLOAK_APPLY" ]
then
cd /gitops/terraform/keycloak
echo "applying keycloak terraform"
terraform init 
terraform destroy -auto-approve  #!* todo need to --> apply
echo "keycloak terraform complete"
else
  echo "skipping keycloak terraform because SKIP_KEYCLOAK_APPLY is set"
fi

export GITLAB_BASE_URL=https://gitlab.<AWS_HOSTED_ZONE_NAME>
export GITLAB_TOKEN=gQevK69TPXSos5cXYC7m

# todo
# curl --request POST \
# --url "https://gitlab.<AWS_HOSTED_ZONE_NAME>/api/v4/projects/6/hooks" \
# --header "content-type: application/json" \
# --header "PRIVATE-TOKEN: gQevK69TPXSos5cXYC7m" \
# --data '{
#   "id": "6",
#   "url": "https://atlantis.<AWS_HOSTED_ZONE_NAME>/events",
#   "token": "c75e7d48b854a36e13fb1d76a6eb5aa750a5e83a3ec6a0093413ed71d3313622",
#   "push_events": "true",
#   "merge_requests_events": "true",
#   "enable_ssl_verification": "true"
# }'


# apply terraform
if [ -z "$SKIP_GITLAB_APPLY" ]
then
  cd /gitops/terraform/gitlab
  echo "applying gitlab terraform"
  terraform init 
  terraform destroy -auto-approve  #!* todo need to --> apply
  echo "gitlab terraform complete"
else
  echo "skipping gitlab terraform because SKIP_GITLAB_APPLY is set"
fi

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
  cd /gitops/terraform/cypress
  npm ci
  $(npm bin)/cypress run
  cd /gitops/terraform

  export GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN=$(cat ./.gitlab-bot-access-token)
  export RUNNER_REGISTRATION_TOKEN=$(cat ./.gitlab-runner-registration-token)

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

  TF_OUTPUT=$(terraform output -json)


  GITLAB_KUBEFIRST_GROUP_ID=$(curl -s --request POST --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" --header "Content-Type: application/json" \
    --data "{\"path\": \"kubefirst\", \"name\": \"kubefirst\" }" \
    "https://$GITLAB_URL/api/v4/groups" | jq -r .id)


  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=ACM_CERTIFICATE_ARN" --form "value=$(echo $TF_OUTPUT | jq -r .acm_arn.value)"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=ECR_REGISTRY_BASE_URL" --form "value=$(echo $TF_OUTPUT | jq -r .ecr_repo_arns.value.metaphor_repository_info.registry_url | cut -d/ -f1)" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=AWS_ACCESS_KEY_ID" --form "value=$AWS_ACCESS_KEY_ID" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=AWS_SECRET_ACCESS_KEY" --form "value=$AWS_SECRET_ACCESS_KEY" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=AWS_ACCOUNT_ID" --form "value=$(aws sts get-caller-identity | jq -r .Account)" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" --form "value=$GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=GITLAB_BOT_SSH_PRIVATE_KEY" --form "value=$(cat ${HOME}/.ssh/id_rsa)" > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=GITLAB_BOT_SSH_PUBLIC_KEY" --form "value=$(cat ${HOME}/.ssh/id_rsa.pub)"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=HOSTED_ZONE_NAME" --form "value=$HOSTED_ZONE_NAME"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=GITLAB_KUBEFIRST_GROUP_ID" --form "value=$GITLAB_KUBEFIRST_GROUP_ID"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=AWS_DEFAULT_REGION" --form "value=$AWS_DEFAULT_REGION"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=RUNNER_REGISTRATION_TOKEN" --form "value=$RUNNER_REGISTRATION_TOKEN"  > /dev/null

  curl -s --header "PRIVATE-TOKEN: $GITLAB_ROOT_USER_PERSONAL_ACCESS_TOKEN" \
  --request POST "https://$GITLAB_URL/api/v4/groups/$GITLAB_KUBEFIRST_GROUP_ID/variables" \
  --form "key=GITLAB_BOT_ROOT_PASSWORD" --form "value=$GITLAB_BOT_ROOT_PASSWORD"  > /dev/null
fi


echo
echo
echo 
echo 
echo
echo
echo "    congratulations you've made it."
echo "    so what next? checkout our docs!"
echo "       https://docs.kubefirst.com/starter"
echo
echo
echo
echo "    tl;dr"
echo
echo "      1. visit your new GitLab instance at"
echo "           https://$GITLAB_URL/kubefirst"
echo "      2. sign in with:"
echo "           username: root"
echo "           password: $GITLAB_BOT_ROOT_PASSWORD"
echo "      3. import the metaphor project by repo url to your new kubefirst group in gitlab"
echo "           repo url: https://github.com/kubefirst/metaphor.git"
echo "      4. commit to the master branch of metaphor and checkout your pipelines"
echo "         https://$GITLAB_URL/kubefirst/metaphor/-/pipelines"
echo "           app url: metaphor-development.$HOSTED_ZONE_NAME"
echo "      5. We created an S3 bucket to be the source of truth and state store of your kubefirst"
echo "         starter plan, its name is $S3_BUCKET_NAME"
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
