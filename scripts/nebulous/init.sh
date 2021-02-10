#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/init.sh
###

set -e

source ~/.profile

# TODO BEFORE EXECUTIONS
# echo
# echo
# echo "EXITING INTENTIONALLY"
# echo "THIS SHOULD TOTALLY WORK"
# exit 0
# echo
# echo
#* need to tokenize terraform/main.tf
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
echo "      25 - 30 min to provision your new aws infrastructure. while you're waiting"
echo "      we recommend checking out our docs to familiarize yourself with what's ahead"
echo
echo "      https://docs.kubefirst.com/starter"
echo
echo
echo
echo

sleep 18 

echo "executing source-profile.sh"
source /scripts/nebulous/source-profile.sh 

ssh-keygen -o -t rsa -b 4096 -C "${EMAIL_ADDRESS}" -f $HOME/.ssh/id_rsa -q -N "" > /dev/null

echo "copying ssh keys to terraform/terraform-ssh-key*"
cp ~/.ssh/id_rsa /terraform/terraform-ssh-key
cp ~/.ssh/id_rsa.pub /terraform/terraform-ssh-key.pub
sleep 2

# setup environment variables
HOSTED_ZONE_NAME=$(aws route53 get-hosted-zone --id "${AWS_HOSTED_ZONE_ID}" | jq -r .HostedZone.Name)
HOSTED_ZONE_NAME=${HOSTED_ZONE_NAME%?}
EMAIL_DOMAIN=$(echo $EMAIL_ADDRESS |  cut -d"@" -f2)
BUCKET_NAME=kubefirst-demo-$(openssl rand -hex 15)
AWS_ACCOUNT_ID=$(aws sts get-caller-identity | jq -r .Account)
IAM_USER_ARN=$(aws sts get-caller-identity | jq -r .Arn)
GITLAB_URL_PREFIX=gitlab-kubefirst-v1
GITLAB_URL="${GITLAB_URL_PREFIX}.${HOSTED_ZONE_NAME}"
GITLAB_BOT_ROOT_PASSWORD=$(openssl rand -hex 11)
GITLAB_ROOT_USER=root

#* terraform separation: all these values should come from pre-determined env's
export TF_VAR_aws_account_id=$AWS_ACCOUNT_ID
export TF_VAR_hosted_zone_name=$HOSTED_ZONE_NAME
export TF_VAR_hosted_zone_id=$AWS_HOSTED_ZONE_ID
export TF_VAR_gitlab_url=$GITLAB_URL
export TF_VAR_email_domain=$EMAIL_DOMAIN
export TF_VAR_region=$AWS_DEFAULT_REGION
export TF_VAR_iam_user_arn=$IAM_USER_ARN

if [[ "$AWS_DEFAULT_REGION" == "us-east-1" ]]; then
  S3_BUCKET_NAME=$(aws s3api create-bucket --bucket $BUCKET_NAME --region $AWS_DEFAULT_REGION | jq -r .Location | cut -d/ -f2 )
else
  S3_BUCKET_NAME=$(aws s3api create-bucket --bucket $BUCKET_NAME --region $AWS_DEFAULT_REGION --create-bucket-configuration LocationConstraint=$AWS_DEFAULT_REGION | jq -r .Location | cut -d/ -f3 | cut -d. -f1 )
fi

if [[ "$S3_BUCKET_NAME" == "" ]]; then
  echo "error creating the s3 bucket! we can't go on, check your credentials, region, etc and try again"
  exit 1
fi
echo
echo
echo
echo "your s3 bucket name is:"
echo $S3_BUCKET_NAME
echo
echo
sleep 3

sed -i "s|@S3_BUCKET_NAME@|${S3_BUCKET_NAME}|g" "/terraform/main.tf"
sed -i "s|@AWS_DEFAULT_REGION@|${AWS_DEFAULT_REGION}|g" "/terraform/main.tf"

cd terraform

terraform init 

terraform apply -auto-approve

echo
echo "testing gitlab for connectivity"
echo
/scripts/nebulous/wait-for-200.sh "https://${GITLAB_URL}/help"
echo
echo "gitlab is ready, executing cypress"
echo


#! uncomment these lines
export CYPRESS_BASE_URL="https://${GITLAB_URL}"
export CYPRESS_gitlab_bot_username_before=$GITLAB_ROOT_USER
export CYPRESS_gitlab_bot_password=$GITLAB_BOT_ROOT_PASSWORD
cd cypress
# ## remove any .skip references in cypress/cypress/integration/gitlab-init.spec.js
npm ci
$(npm bin)/cypress run
cd .. 

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


echo "getting ~/kube/config for eks access"
K8S_CLUSTER_NAME=$(terraform output -json | jq -r .cluster_name.value)
aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name $K8S_CLUSTER_NAME
chmod 0600 ~/.kube/config

echo
echo "creating applications namespaces"
kubectl apply -f /scripts/nebulous/kubernetes/namespaces.yaml

echo
echo "adding helm repositories"
helm repo add gitlab https://charts.gitlab.io > /dev/null
helm repo add bitnami https://charts.bitnami.com/bitnami > /dev/null
helm repo update > /dev/null
echo
echo
echo "modifying heelm values yamls"
echo
echo "detokenizing values.yaml files"
/scripts/ci-cd/detokenize.sh "@HOSTED_ZONE_NAME@" "${HOSTED_ZONE_NAME}" "/scripts/nebulous/helm"
/scripts/ci-cd/detokenize.sh "@GITLAB_URL@" "https://${GITLAB_URL}" "/scripts/nebulous/helm"
/scripts/ci-cd/detokenize.sh "@RUNNER_REGISTRATION_TOKEN@" "${RUNNER_REGISTRATION_TOKEN}" "/scripts/nebulous/helm"
echo
echo

echo "installing gitlab runners to build and deploy metaphor"
echo
helm -n gitlab-runner install gitlab-runner -f /scripts/nebulous/helm/gitlab-runner.yaml gitlab/gitlab-runner
kubectl -n gitlab-runner scale deployment/gitlab-runner-gitlab-runner --replicas=2
echo
echo
echo "installing external-dns for automatic DNS records"
echo
helm -n external-dns install external-dns -f /scripts/nebulous/helm/external-dns.yaml bitnami/external-dns
echo
echo

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
echo "         starter plan, its name is $S3_BUCKET_NAMEE"
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

