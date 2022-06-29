# nebulous
The Kubefirst Open Source Platform

![images/nebulous-arch.png](images/nebulous-arch.png)

## tl;dr:
- step 1: establish a new aws account with a single hosted zone that's configured to receive traffic from your domain name registrar
- step 2: add your 6 configuration values to kubefirst.env and run the nebulous container
- step 3: get a fully-functioning application delivery ecosystem, complete with kubernetes, gitops, vault, terraform, atlantis, gitlab, gitlab-runner, and a sample app that demonstrates how it all works.

---

# user guide

## docs
- [introduction](https://docs.kubefirst.com/)
- [installation](https://docs.kubefirst.com/nebulous/install.html)
- [getting familiar](https://docs.kubefirst.com/kubefirst/getting-started.html)
- [teardown](https://docs.kubefirst.com/nebulous/teardown.html)
- [faq](https://docs.kubefirst.com/nebulous/faq.html)
- [contact](https://docs.kubefirst.com/contact.html)

---

# contributor guide

The docs above are tailored to our end user's experience. However things are a little different if you're contributing to nebulous itself. The docs that follow are intended only for source contributors.

### step 1 - setup nebulous.env

For a first run, this step is no different than the guidance to our end users, you need to set up a `kubefirst.env` in the nebulous repo's root directory. You can create the file template by running this from your terminal, editing with your values with the normal settings.

For subsequent executions, especially while debugging, it's sometimes helpful to use some additional environment variables that allow you to control the flow of execution. See the notes in each section for details on controlling your debugging.

In addition to the flow controls, you'll also find some hack comments by the various terraform apply commands. This allows you to change apply commands to exiting deploy commands. This can also be valuable when you need a mulligan on a particular section.

```bash
cat << EOF > kubefirst.env
###############################
# Access settings
# The AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY are your credentials to 
# log into your AWS account, you can often find these in `~/.aws/credentials`
# The AWS_DEFAULT_REGION is the aws region that your new infrastructure will provision in - 
# The AWS_HOSTED_ZONE_NAME is the domain name associated with your prerequesite hosted zone in route53 - it should look similar to yourdomain.com with no www. prefix and no . suffix

AWS_ACCESS_KEY_ID=YOUR_ADMIN_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_ADMIN_AWS_SECRET_ACCESS_KEY
AWS_HOSTED_ZONE_NAME=yourdomain.com
AWS_DEFAULT_REGION=us-east-2


###################
# Admin settings
# The EMAIL_ADDRESS is used for the ssh key that's generated and for certificate expiration notifications
# The GITLAB_BOT_ROOT_PASSWORD is the password to use for the gitlab root user, change this to a value only you know

EMAIL_ADDRESS=YOUR_EMAIL_ADDRESS@yourdomain.com
GITLAB_BOT_ROOT_PASSWORD=123456ABCDEF!


###############################
# Users:
# The BUCKET_RAND needs to be set and uncommented before destroy, see the teardown 
# docs for details.
# 
# Contributors: 
# The BUCKET_RAND has implications on bucket reuse when iterating
# once you successfully get past base terraform apply, 
# take the random suffix that was generated, apply it to the 
# next line, and start reusing the bucket for subsequent runs.
# if you don't set this value on subsequent runs, it will keep 
# generating new buckets for you. You can find this value in the 
# nebulous execution output.
# 
# BUCKET_RAND=abc123


###############################
# Note: Operational Flow Controls - uncomment the items below 
# when you want to skip over various sections. Leaving them
# all commented like they are here will execute everything.
# 
#
# SKIP_HZ_CHECK=true
# SKIP_DETOKENIZATION=true
# SKIP_BASE_APPLY=true
# SKIP_GITLAB_RECONFIG=true
# SKIP_GITLAB_APPLY=true
# SKIP_ARGOCD_APPLY=true
# SKIP_VAULT_APPLY=true
# SKIP_SSH_STORAGE=true
# SKIP_USERS_APPLY=true
# SKIP_OIDC_PATCHING=true

EOF
```

### step 2 - build nebulous locally

Come up with local tag name for your nebulous image. We'll use `foo` as our example local tag name in these docs. To build the `foo` tag of nebulous run the following from your local nebulous repo root directory.

```bash
docker build . -t nebulous:foo
```

### step 3 - running nebulous

Once you have built the `nebulous:foo` image as shown above, you can kickoff the automated init script by running the following. The difference between this guidance and the end user guidance is that this mounts the `gitops`, `scripts`, and `git` directories to your localhost volume so you can negotiate changes to the runtime environment on the fly.

This is how you run the container with the volume mounts. Run this from your nebulous directory:
```
docker run -it --env-file=kubefirst.env -v $PWD/gitops:/gitops -v $PWD/metaphor:/metaphor -v $PWD/scripts:/scripts -v $PWD/git:/git --entrypoint /scripts/nebulous/init.sh nebulous:foo
```

### step 4 - teardown (once you're ready to tear it all back down, obviously)

There are a few things to note about teardown.

Nebulous creates a VPC, some subnets, a gitlab server, a kubernetes cluster, some policies, roles, and a few other things (complete list in the teardown docs). Terraform knows about all of these things, and if you only created these resources, you'll be able to run teardown without thinking too hard.

However, terraform is only able to destroy resources that are managed in terraform. It doesn't know about things you do manually. Anything you may have added through non-terraform operations must be manually removed before running the teardown script. 

Let's consider, for example, a scenario where you manually `helm install`ed an app to your new cluster, and that app spins up a new load balancer in your VPC. If you don't remove that app and its load balancer before running destroy, you won't be able to complete the terraform destroy operation. This is because you can't remove a VPC that still has a live load balancer running in it.

With that context in mind, once you've removed the manual things you may have added to this environment, you can kickoff the automated destroy script by running:
```
docker run -it --env-file=kubefirst.env -v $PWD/gitops:/gitops -v $PWD/metaphor:/metaphor -v $PWD/scripts:/scripts -v $PWD/git:/git --entrypoint /scripts/nebulous/destroy.sh  nebulous:foo  
```

### New README

# Flare

- [Flare](#flare)
  - [Start](#start)
    - [Start Environment variables](#start-environment-variables)
    - [Start Actions](#start-actions)
    - [Start Confirmation](#start-confirmation)
  - [Destroy](#destroy)
    - [Destroy Actions](#destroy-actions)
      - [Notes:](#notes)

## Start

### Start Environment variables

In order to start Kubefirst, the required environment variables are:

| Variable         | example            |
|------------------|--------------------|
| AWS_PROFILE      | default            |
| AWS_REGION       | us-east-1          |
| HOSTED_ZONE_NAME | mydomain.com       |
| ADMIN_EMAIL      | myemail@somewhere.com |

### Start Actions

```bash
touch ~/.flare
mkdir -p ~/.kubefirst
cd ~/git/kubefirst/gitlab/flare # change to your dir if different
go build -o bin/flare main.go
./bin/flare nebulous init --admin-email $ADMIN_EMAIL --cloud aws --hosted-zone-name $HOSTED_ZONE_NAME --region $AWS_REGION
./bin/flare nebulous create
```

### Start Confirmation

```bash
aws eks update-kubeconfig --name kubefirst
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
kubectl -n argocd port-forward svc/argocd-server 8080:80
```

## Destroy

To destroy remote then local.

These environment variables are expected:

| Variable         | example                                                                                       |
|------------------|-----------------------------------------------------------------------------------------------|
| AWS_PROFILE      | default                                                                                       |
| AWS_REGION       | us-east-1                                                                                     |
| AWS_ACCOUNT_ID   | 1xxxxxxxxxx4                                                                                  |
| HOSTED_ZONE_NAME | mydomain.com                                                                                  |
| GITLAB_TOKEN     | "xxxxx1-xx1x-x1xx-1" # replace with value from ~/.flare (only needed if you got to gitlab tf) |


### Destroy Actions
```bash
./bin/flare nebulous destroy
rm -rf ~/.kubefirst
rm ~/.flare
```

#### Notes:

added gitlab.yaml to registry
pushing local to soft origin

