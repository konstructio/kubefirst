docker run --env-file=kubefirst.env -v $PWD/terraform:/terraform -v $PWD/scripts:/scripts --entrypoint /scripts/nebulous/init.sh nebulous:foo

```yaml
apiVersion: "kubernetes-client.io/v1"
kind: ExternalSecret
metadata:
  name: kubefirst-secrets
spec:
  backendType: vault
  vaultMountPoint: kubernetes/starter-us-east-1
  vaultRole: external-secrets
  kvVersion: 2
  data:
  - name: AWS_ACCESS_KEY_ID
    key: secret/data/kubefirst
    property: AWS_ACCESS_KEY_ID
  - name: AWS_SECRET_ACCESS_KEY
    key: secret/data/kubefirst
    property: AWS_SECRET_ACCESS_KEY
  - name: AWS_HOSTED_ZONE_ID
    key: secret/data/kubefirst
    property: AWS_HOSTED_ZONE_ID
  - name: AWS_ACCOUNT_ID
    key: secret/data/kubefirst
    property: AWS_ACCOUNT_ID
  - name: AWS_DEFAULT_REGION
    key: secret/data/kubefirst
    property: AWS_DEFAULT_REGION
  - name: EMAIL_ADDRESS
    key: secret/data/kubefirst
    property: EMAIL_ADDRESS
  - name: VAULT_ADDR
    key: secret/data/kubefirst
    property: VAULT_ADDR
  - name: VAULT_TOKEN
    key: secret/data/kubefirst
    property: VAULT_TOKEN
  - name: GITLAB_BOT_ROOT_PASSWORD
    key: secret/data/kubefirst
    property: GITLAB_BOT_ROOT_PASSWORD

```

# todos 
- need to add `argocd app wait/sync` after each sync wave, potentially add kuttl tests
- new builder / nebulous image with vault-cli (see `kubefirst-builder:spike`) in jobs and figure out whats missing or what was published


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

This step is actually no different than the guidance to our end users, you need to set up a `kubefirst.env` in the nebulous repo's root directory. You can create the file template by running this from your terminal, editing with your values for these 5 settings.

```
cat << EOF > kubefirst.env
AWS_ACCESS_KEY_ID=YOUR_ADMIN_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_ADMIN_AWS_SECRET_ACCESS_KEY
AWS_HOSTED_ZONE_ID=YOUR_AWS_HOSTED_ZONE_ID
AWS_DEFAULT_REGION=YOUR_AWS_REGION
EMAIL_ADDRESS=YOUR_EMAIL_ADDRESS
GITLAB_BOT_ROOT_PASSWORD=YOUR_GITLAB_BOT_ROOT_PASSWORD
EOF
```

### step 2 - build nebulous locally

Come up with local tag name for your nebulous image. We'll use `foo` as our example local tag name in these docs. To build the `foo` tag of nebulous run the following from your local nebulous repo root directory.

```bash
docker build . -t nebulous:foo
```

### step 3 - running nebulous

Once you have built the `nebulous:foo` image as shown above, you can kickoff the automated init script by running

```
docker run --env-file=kubefirst.env -v $PWD/terraform:/terraform --entrypoint /scripts/nebulous/init.sh nebulous:foo
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