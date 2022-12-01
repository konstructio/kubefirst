# mkdocs

## running docs on your localhost

1. start the mkdocs-material container from your repo root directory:

```shell
# run the following from the repo root
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```

2. click <http://localhost:8000>
3. edit your Markdown documentation in your favorite editor and get realtime feedback

## publishing to preprod (temp docs)

Once the modifications are merged in the `main` branch, a [GitHub Action](https://github.com/kubefirst/kubefirst/blob/main/.github/workflows/publish-docs.yaml) will build, and publish it automatically on the [documentation prepod site](https://docs.kubefirst.com/preprod/). Please ask [@fharper](https://github.com/fharper) for access.

## promote to prod

Once you are happy with your changes in prepod, you need to manually run the [Promote Docs To Prod GitHub Action](https://github.com/kubefirst/kubefirst/actions/workflows/promote-docs-to-prod.yaml).
