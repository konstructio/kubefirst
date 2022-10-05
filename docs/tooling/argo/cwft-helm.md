# CWFT-HELM

A set of [Templates](https://github.com/kubefirst/gitops-template/blob/main/components/argo-cwfts/cwft-helm.yaml) used manipulate helm artifacts


## helm-check-chart-museum

This function is meant to check if the [Chart Museum](https://chartmuseum.com/) deployed on your installations is already available to receive new artifacts.

It is meant to be use before steps that will try to deploy new artifacts on a fresh cluster deployed. 

## helm-publish-chart

This function is meant to publish a chart to a helm repository, such as the [Chart Museum](https://chartmuseum.com/) deployed on your installations is already available to receive new artifacts.

## helm-set-chart-version

This function is used to set a version of chart to be pushed. 

## helm-set-chart-versions

This function is used to set a versions of chart to be pushed. 

## helm-get-chart-release-from-version

This function returns a release string from a chart version. 


## helm-get-chart-version

This function returns a chart version. 

## helm-set-environment-version

This function set version as enviroment variable

## helm-increment-chart-patch

This function is used to bump a chart version. The logic for automatic bumps. 
