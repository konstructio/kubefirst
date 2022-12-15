# Contributing to Kubefirst

Firstly, we want to thank you for investing your precious time to contribute to Kubefirst!

_⚠️ Please note that this file is a work-in-progress, so more details will be added in the future._

Note we have a [code of conduct](CODE_OF_CONDUCT.md) which needs to be followed in all your interactions with the project to keep our community healthy.

## Ways to Contribute

At Kubefirst, we believe that every contribution are valuable, not just the code one, which means we welcome

- [bug reports](https://github.com/kubefirst/kubefirst/issues/new);
- [feature requests](https://github.com/kubefirst/kubefirst/issues/new?assignees=&labels=feature-request&template=feature_request.md&title=);
- [documentations issues reports](https://github.com/kubefirst/kubefirst/issues/new?assignees=&labels=docs&template=docs.yml&title=%5BDocs%5D%3A+) like unclear section, missing information or even typos;
- and, of course, any code contributions to Kubefirst, or the documentations.

Before making a code change, first discuss your idea via an [issue](https://github.com/kubefirst/kubefirst/issues/new/choose). Please check if a feature request or bug report does [already exist](https://github.com/kubefirst/kubefirst/issues/) before creating a new one.

## Getting Started with the Code

Kubefirst is created using the [Go Programming Language](https://go.dev). To set up your computer, follow [these steps](https://go.dev/doc/install).

Once Go is installed, you can run Kubefirst from any branch using `go run .`. Go will automatically install the needed modules listed in the [go.mod](go.mod) file. As an example, if you want to create a [local cluster](https://docs.kubefirst.io/kubefirst/local/install.html), the command would be `go run . local`.
b
Since Go is a compiled programming language, every time you use the `run` command, Go will compile the code before running it. If you want to save time, you can compile your code using `go build`, which will generate a file named `kubefirst`. You will then be able to run your compiled version with the `./kubefirst` command.

## Help

If you need help in your Kubefirst journey as a contributor, please join our [Slack Community](http://kubefirst.io/slack). We have the `#kubefirst-oss` channel where you can ask any questions or get help with anything contribution-related. For support as an user, please ask in the `#helping-hands` channel, or directly to @fharper (Fred in Slack), our Developer Advocate.
