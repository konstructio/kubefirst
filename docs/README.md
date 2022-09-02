# mkdocs

### running docs on your localhost

1. start the mkdocs-material container from your repo root directory:
```
# run the following from the repo root
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```
2. click http://localhost:8000
3. edit your markdown documentation in your favorite editor and get realtime feedback

### publishing to preprod (temp docs)

```
# confirm aws config pointed at mgmt account

pip3 install mkdocs mkdocs-material
rm -rf ./dist
mkdocs build --no-directory-urls -d dist
aws s3 sync dist s3://docs.kubefirst.com/preprod --delete
aws cloudfront create-invalidation --distribution-id E1DXWJ8ITAKV61 --paths "/preprod/*"
# https://docs.kubefirst.com/preprod/index.html
```

### promote to prod
```
aws s3 sync dist s3://docs.kubefirst.com --delete
aws s3 sync dist s3://docs.kubefirst.com/preprod --delete # prod deploy blows away preprod
aws cloudfront create-invalidation --distribution-id E1DXWJ8ITAKV61 --paths "/*"
# https://docs.kubefirst.com/index.html
```