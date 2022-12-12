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

merging docs changes to the main branch will automatically kick off a publish to preprod using the [Publish Docs](https://github.com/kubefirst/kubefirst/actions/workflows/publish-docs.yaml) action.
[https://docs.kubefirst.com/preprod/index.html](https://docs.kubefirst.com/preprod/index.html)


### promote to prod

after confirming there are no rendering issues in preprod, run the github action [Promote Docs To Prod](https://github.com/kubefirst/kubefirst/actions/workflows/promote-docs-to-prod.yaml) to update the live site.
[https://docs.kubefirst.com/index.html](https://docs.kubefirst.com/index.html)