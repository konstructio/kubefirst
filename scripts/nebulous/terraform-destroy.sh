#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/terraform-destroy.sh
###

set -e

K8S_CLUSTER_NAME=kubefirst

aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name $K8S_CLUSTER_NAME
chmod 0600 ~/.kube/config

/scripts/nebulous/cleanup-cluster.sh

HOSTED_ZONE_NAME=$(aws route53 get-hosted-zone --id "${AWS_HOSTED_ZONE_ID}" | jq -r .HostedZone.Name | cut -d"." -f-3)
EMAIL_DOMAIN=$(echo $EMAIL_ADDRESS |  cut -d"@" -f2)
BUCKET_NAME=kubefirst-demo-$(openssl rand -hex 15)
AWS_ACCOUNT_ID=$(aws sts get-caller-identity | jq -r .Account)
IAM_USER_ARN=$(aws sts get-caller-identity | jq -r .Arn)
GITLAB_URL_PREFIX=gitlab
GITLAB_URL="${GITLAB_URL_PREFIX}.${HOSTED_ZONE_NAME}"
GITLAB_ROOT_USER=root

#* terraform separation: all these values should come from pre-determined env's
export TF_VAR_aws_account_id=$AWS_ACCOUNT_ID
export TF_VAR_hosted_zone_name=$HOSTED_ZONE_NAME
export TF_VAR_hosted_zone_id=$AWS_HOSTED_ZONE_ID
export TF_VAR_gitlab_url=$GITLAB_URL
export TF_VAR_email_domain=$EMAIL_DOMAIN
export TF_VAR_region=$AWS_DEFAULT_REGION
export TF_VAR_iam_user_arn=$IAM_USER_ARN
export TF_VAR_gitlab_bot_root_password=$GITLAB_BOT_ROOT_PASSWORD


echo "tearing down all the infrastructure provisioned by nebulous"

cd /terraform

terraform init

terraform destroy --auto-approve