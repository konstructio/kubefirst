#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/terraform-destroy.sh "us-east-1" "k8s-preprod"
###

set -e

AWS_DEFAULT_REGION=$1
K8S_CLUSTER_NAME=$2

aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name k8s-preprod $K8S_CLUSTER_NAME

/scripts/nebulous/cleanup-cluster.sh

echo "tearing down all the infrastructure provisioned by nebulous"

cd terraform

terraform destroy --auto-approve