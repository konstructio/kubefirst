# mkdocs-material & mkdocs

`mkdocs-material` implements `mkdocs` in a really straightforward way with a great local and production experience. It powers the site that you're reading here. It's a single git repository and all the docs are written in markdown. It has a simple table of contents structure and offers a simple and exceptional localhost experience. It also boasts a fully functional search without any additional components required.

## Running Locally

Start the mkdocs-material container from your mkdocs repo root directory:
```
docker run --rm -it -p 8000:8000 -v ${PWD}:/docs squidfunk/mkdocs-material
```

View the local docs by visiting [http://localhost:8000](http://localhost:8000)

Edit your markdown documentation in your favorite editor and get realtime feedback in the UI

## Deploying Updates

To deploy your updates to your live docs, merge request your changes to master. Once merged a pipeline will deliver your changes to preprod and production.

## Managing the Table of Contents

In the mkdocs repo root, you'll find how we're managing our navigation, site, and theme in a simple yaml layout. This is all that's needed to add additional markdown pages to your site.
```
nav:
  - Home:
    - Welcome: 'index.md'
    - Getting Started:
      - Overview: 'getting-started/overview.md'
      - Gitlab Repositories: 'getting-started/gitlab-repositories.md'
    - Open Source:
      - Credit: 'kubefirst/credit.md'
    ...
```

## Search

Search results are rendered automatically from the static site content without any additional resources in play, it's like magic.

## Expanding the Site

See [https://squidfunk.github.io/mkdocs-material/](https://squidfunk.github.io/mkdocs-material/) for more details on mkdocs-material features and releases.
