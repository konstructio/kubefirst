# CWFT-GIT

A set of [Templates](https://github.com/kubefirst/gitops-template/blob/main/components/argo-cwfts/cwft-git.yaml) used checkout and push changes to git repos agnostic of the provider. 

Note: When mentioned `gitops` repo, this is the repo on your installation with tooling and gitops state. So, when handling repos you usually have a `gitops` that is the state of your cluster and an `application` repo that may want to change the state of the cluster during its installations steps. 

##  git-short-sha

It returns the `short-sha` of a commit to be used on later steps to identify artifacts or jobs. 

## SSH Auth Based Functions

Used by our github installations. 

### git-checkout-with-gitops-ssh
Used to checkout an `application` repo and `gitops` repo using SSH keys, used for example on our github type instalations. 

### git-commit-ssh

Used to `commit` and `push` a change against an repo. It has retrial strategy to allow to work well when parallel jobs may be running and trying to commit an update of state. 

It uses SSH keys to authenticate to push the changes. 


## HTTPS Auth Based Functions

Used by our gitlab installations. 

### git-checkout-with-gitops
Used to checkout an `application` repo and `gitops` repo using HTTPS and a Token, used for example on our gitlab type instalations. 

### git-commit

Used to `commit` and `push` a change against an repo. It has retrial strategy to allow to work well when parallel jobs may be running and trying to commit an update of state. 

It uses  HTTPS and a Token to authenticate to push the changes. 

