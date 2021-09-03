docker run --env-file=kubefirst.env -v $PWD/terraform:/terraform -v $PWD/scripts:/scripts --entrypoint /scripts/nebulous/init.sh nebulous:foo

# todos to discuss before next execution
- need to add `argocd app wait/sync` after each sync wave, potentially add kuttl tests
- new builder / nebulous image with vault-cli (see `kubefirst-builder:spike`) in jobs and figure out whats missing or what was published
- change gitlab group name to `kubefirst`
- search for `preprod` 
- kubefirst_worker_nodes_role address whether assume role is still needed
- descriptions for all variables
- move `/terraform/security-groups` to `/terraform/vpc/gitlab-sg.tf` ? its only one security group
- update LICENSE
- aws route53 test-dns-answer --hosted-zone-id $HOSTED_ZONE_ID --hosted-zone-id, --record-name, --record-type
- gitops argocd - it's presently scripted
- 


# nebulous
The Kubefirst Open Source Starter Plan repository

![images/starter.png](images/starter.png)

# docs
- [introduction](https://docs.kubefirst.com/starter/)
- [installation](https://docs.kubefirst.com/starter/nebulous/)
- [getting familiar](https://docs.kubefirst.com/starter/getting-familiar/)
- [teardown](https://docs.kubefirst.com/starter/teardown/)
- [faq](https://docs.kubefirst.com/starter/faq/)

---

# contributor guide

The docs above are tailored to our end user's experience. However things are a little different if you're contributing to nebulous itself. The docs that follow are intended only for source contributors.

### step 1 - setup nebulous.env

For a first run, this step is no different than the guidance to our end users, you need to set up a `kubefirst.env` in the nebulous repo's root directory. You can create the file template by running this from your terminal, editing with your values with the normal settings.

For subsequent executions, especially while debugging, it's sometimes helpful to use some additional environment variables that allow you to control the flow of execution. See the notes in each section for details on controlling your debugging.

In addition to the flow controls, you'll also find some hack comments by the various terraform apply commands. This allows you to change apply commands to exiting deploy commands. This can also be valuable when you need a mulligan on a particular section.

```
cat << EOF > kubefirst.env
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
# SKIP_VAULT_APPLY=true
# SKIP_KEYCLOAK_APPLY=true


###############################
# Note: Bucket Reuse - when you successfully get past base terraform
# apply, take the random suffix that was generated, apply it to the 
# next line, and start reusing the bucket for subsequent iterations.
# if you don't set this value on subsequent runs, it will keep 
# generating new buckets for you. You can find this value in the 
# nebulous execution output.
# 
# BUCKET_RAND=abc123

###############################
# AWS Account Information
AWS_ACCESS_KEY_ID=YOUR_ADMIN_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_ADMIN_AWS_SECRET_ACCESS_KEY
AWS_HOSTED_ZONE_ID=YOUR_AWS_HOSTED_ZONE_ID
AWS_DEFAULT_REGION=YOUR_AWS_REGION

AWS_ACCESS_KEY_ID=AKIAR3B33MDMPI6LS3LE
AWS_SECRET_ACCESS_KEY=ode7PKykNIOU9ZZ64psfyQMbp8feFpXCBOHaPEdT
AWS_HOSTED_ZONE_ID=Z0894979RGNIP7HU126B
AWS_HOSTED_ZONE_NAME=starter.kubefirst.com
AWS_DEFAULT_REGION=us-east-1

###############################
# Logistics
EMAIL_ADDRESS=EMAIL_ADDRESS=YOUR_EMAIL_ADDRESS

###############################
# Gitlab
GITLAB_BOT_ROOT_PASSWORD=$omePasswordOnlyYouKn0w!

EOF
```

### step 2 - build nebulous locally

Come up with local tag name for your nebulous image. We'll use `foo` as our example local tag name in these docs. To build the `foo` tag of nebulous run the following from your local nebulous repo root directory.

```bash
docker build . -t nebulous:foo
```

### step 3 - running nebulous

Once you have built the `nebulous:foo` image as shown above, you can kickoff the automated init script by running the following. The difference between this guidance and the end user guidance is that this mounts the scripts and git directories to your localhost volume so you can negotiate changes to either space.

```
docker run -it --env-file=kubefirst.env -v $PWD/gitops:/gitops -v $PWD/scripts:/scripts -v $PWD/git:/git --entrypoint /scripts/nebulous/init.sh nebulous:foo
```

### step 4 - teardown

Once you have built the `nebulous:foo` image as shown above, you can kickoff the automated init script by running

```
docker run -it --env-file=kubefirst.env -v $PWD/terraform:/terraform --entrypoint /bin/sh nebulous:foo
```

and then in your interactice docker shell run

```
/scripts/nebulous/terraform-destroy.sh
```