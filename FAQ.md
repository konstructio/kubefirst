# Overview 

This page is meant to help developers to be introduced to some knowledge we re-use when improve `kubefirst` tooling. 

We apreciate all contribution provided following our [Contributor Covenant Code of Conduct](https://github.com/kubefirst/kubefirst/blob/main/CODE_OF_CONDUCT.md). 

Some points on this page as the state of today in the code and can be improved as we evolve get exposed to more edge scenarios, theay are not to meant to be rules that are enforced, they are more hints of what we used before and worked better to the situation presented to it. 

We will add some links for answers that may be open to opnions so we can evolve our understand of more scenarios possibles and try to learn for that. 

Thanks for participating on our prorject. 


# Questions from Developers / General Coding Patterns

## Golang 

Today, `kubefirst` uses `golang` on most of its components/cli implementation of utilities. 

These are some of the patterns we use: 

### Cobra-cli 

We use [cobra-cli](https://github.com/spf13/cobra) to create our commands, it has its pro/cons.

If you would like to express some opnions on this, we have [this discussion](https://github.com/kubefirst/kubefirst/discussions/531) for it. 

#### How to create a new command? 


This line will add a command under `actionCmd` to create a new `action`. `actionCmd` is special command to be parent of general commands that execute parts of installation or some developers use to test behaviors before creating a function for something. It a nice place to start as a sandbox. 

```bash 
cobra-cli add myCustomCommand -p 'actionCmd'
```

**Tip:** To install it, just run `go install github.com/spf13/cobra-cli@latest` 

## Terraform


## Argo/ArgoCD/Argo Workflows


## Vault


## Atlantis


## Others Tools

## Templates

### Gitops Templates

### Metaphor Templates


### CWFT
