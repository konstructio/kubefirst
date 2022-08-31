# zsh shell utilities

This page documents some zsh shell utilities to improve your localhost's kubernetes experience

### the k alias

Our favorite utility isn't a utility at all, it's a simple alias so that instead of typing the command kubectl, you can kust type the glorious letter k.

To add the k alias to your zsh profile edit your `~/.zshrc` file and add the following line:
```
alias k="kubectl"
```

You can now simply k instead of kubectl
```
k get nodes -owide
k get namespaces
```

### kube-ps1

[https://github.com/jonmosco/kube-ps1](https://github.com/jonmosco/kube-ps1)

`kube-ps1` is a utility that will display your current kubectl context in your terminal

```
brew update
brew install kube-ps1
```

### kubectx and kubens

[https://github.com/ahmetb/kubectx](https://github.com/ahmetb/kubectx)

`kubectx` is a utility to manage and switch between kubectl contexts.

`kubens` is a utility to switch between Kubernetes namespaces.

You get both by installing kubectx

```
brew install kubectx
```

### jq

[https://stedolan.github.io/jq/](https://stedolan.github.io/jq/)

`jq` is like sed for JSON data - you can use it to get values out of json responses.

```
brew update
brew install jq
```
