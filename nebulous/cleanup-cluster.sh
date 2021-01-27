#!/usr/bin/env bash

###
# usage: ./scripts/nebulous/cleanup-cluster.sh
###

set -e

echo
echo "uninstalling the helm deployments"
helm -n external-dns uninstall external-dns
helm -n gitlab-runner uninstall gitlab-runner

echo
echo
echo "IMPORTANT:"
echo "    *NOTE* please delete any kind: Service you've created in the cluster prior to destroying"
echo "    with terraform. the service type: LoadBalancer will be left in your cloud because the ACM"
echo "    certificate will still be in use. this could lead to abandoned resources in your aws account"
echo
echo
sleep 8

echo
echo "deleting development metaphor resources"
kubectl -n development delete service/metaphor deploy/metaphor ingress/metaphor secrets/metaphor-secrets

echo
echo "deleting staging metaphor resources"
kubectl -n staging delete service/metaphor deploy/metaphor ingress/metaphor secrets/metaphor-secrets

echo
echo "deleting staging metaphor resources"
kubectl -n production delete service/metaphor deploy/metaphor ingress/metaphor secrets/metaphor-secrets

echo
echo "deleting cluster namespaces"
kubectl delete namespaces development staging production gitlab-runner external-dns

echo
echo
echo "successfully unintalled metaphor and blew up the namespaces BOOM!"
echo