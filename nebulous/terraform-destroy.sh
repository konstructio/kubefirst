#!/usr/bin/env bash

# todo turn into variables
aws eks update-kubeconfig --region $AWS_DEFAULT_REGION --name k8s-preprod $K8S_CLUSTER_NAME

/scripts/nebulous/cleanup-cluster.sh



echo "tearing down all the infrastructure provisioned by nebulous"

cd terraform-starter

terraform destroy --auto-approve