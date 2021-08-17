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

The docs above are tailored to our end user experience. However things are a little different if you're contributing the nebulous itself. If you're **contributing** to nebulous, the docs below are for you.

### step 1 - setup nebulous.env

This step is actually no different than the guidance to our end users, you need to set up a `kubefirst.env` in the nebulous repo's root directory. You can create the file template by running this from your terminal, editing with your values for these 5 settings.

```
cat << EOF > kubefirst.env
AWS_ACCESS_KEY_ID=YOUR_ADMIN_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_ADMIN_AWS_SECRET_ACCESS_KEY
AWS_HOSTED_ZONE_ID=YOUR_AWS_HOSTED_ZONE_ID
AWS_DEFAULT_REGION=YOUR_AWS_REGION
EMAIL_ADDRESS=YOUR_EMAIL_ADDRESS
EOF
```

### step 2 - build nebulous locally

Come up with local tag name for your nebulous image. We'll use `foo` as our example local tag name in these docs. To build the `foo` tag of nebulous run the following from your local nebulous repo root directory.

```bash
nebulous docker build . -t nebulous:foo
```

### step 3 - running nebulous

Once you have built the `nebulous:foo` image as shown above, you can kickoff the automated init script by running

```
nebulous docker run --env-file=kubefirst.env -v $PWD/terraform:/terraform --entrypoint /scripts/nebulous/init.sh nebulous:foo
```