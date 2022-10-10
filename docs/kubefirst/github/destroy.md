# Teardown

## Step 1: `~/.kubefirst`
If you just recently ran install from your localhost, you'll already have the file on your localhost at `~/.kubefirst` that's needed to destroy. If you don't have this file locally, you'll need to download it from your s3 bucket that was created during provisioning and add it to your home directory.

## Step 2: `Destroy`

With your ~/.kubefirst file in place, run:

```bash
kubefirst cluster destroy
```
