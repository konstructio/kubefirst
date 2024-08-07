# Contributing to Kubefirst

Firstly, we want to thank you for investing your valuable time to contribute to Kubefirst!

_⚠️ Please note that this file is a work-in-progress, so more details will be added in the future._

Note we have a [code of conduct](CODE_OF_CONDUCT.md) which needs to be followed in all your interactions with the project to keep our community healthy.

## Ways to Contribute

At Kubefirst, we believe that every contribution is valuable, not just the code one, which means we welcome

- [bug reports](https://github.com/kubefirst/kubefirst/issues/new);
- [feature requests](https://github.com/kubefirst/kubefirst/issues/new?assignees=&labels=feature-request&template=feature_request.md&title=);
- [documentations issues reports](https://github.com/kubefirst/kubefirst/issues/new?assignees=&labels=feature-request&template=feature_request.md&title=) like unclear section, missing information or even typos;
- and, of course, any code contributions to Kubefirst, or the documentations.

Before making a code change, first discuss your idea via an [issue](https://github.com/kubefirst/kubefirst/issues/new/choose). Please check if a feature request or bug report does [already exist](https://github.com/kubefirst/kubefirst/issues/) before creating a new one.

## Getting Started with the Code

### Dev containers

A [.devcontainer](https://containers.dev/) configuration is provided to allow for a full-featured development environment.

### Local development

Kubefirst is created using the [Go Programming Language](https://go.dev). To set up your computer, follow [these steps](https://go.dev/doc/install).

Once Go is installed, you can run Kubefirst from any branch using `go run .`. Go will automatically install the needed modules listed in the [go.mod](go.mod) file. As an example, if you want to create a [local cluster](https://docs.kubefirst.io/kubefirst/local/install.html), the command would be `go run . k3d create`. Note that even if you run kubefirst from `main`, the [gitops-template](https://github.com/kubefirst/gitops-template) version used will be the [latest release](https://github.com/kubefirst/gitops-template/releases). If you also want to use the latest from `main` for the template also, you need to run to use the `--gitops-template-url`, and the `--gitops-template-branch` as follow:

```shell
go run . k3d create --gitops-template-url https://github.com/kubefirst/gitops-template --gitops-template-branch main
```

Since Go is a compiled programming language, every time you use the `run` command, Go will compile the code before running it. If you want to save time, you can compile your code using `go build`, which will generate a file named `kubefirst`. You will then be able to run your compiled version with the `./kubefirst` command.

## Getting Started with the Documentation

Please check the [CONTRIBUTING.md](https://github.com/kubefirst/docs/blob/main/CONTRIBUTING.md) file from the [docs](https://github.com/kubefirst/docs/) repository.

## Help

If you need help in your Kubefirst journey as a contributor, please join our [Slack Community](http://kubefirst.io/slack). We have the `#contributors` channel where you can ask any questions or get help with anything contribution-related. For support as a user, please ask in the `#helping-hands` channel, or directly to @fharper (Fred in Slack), our Developer Advocate.
