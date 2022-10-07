# Overview 

This page is meant to help developers to be introduced to some knowledge we re-use when improve `kubefirst` tooling. 

We apreciate all contribution provided following our [Contributor Covenant Code of Conduct](https://github.com/kubefirst/kubefirst/blob/main/CODE_OF_CONDUCT.md). 

Some points on this page as the state of today in the code and can be improved as we evolve get exposed to more edge scenarios, theay are not to meant to be rules that are enforced, they are more hints of what we used before and worked better to the situation presented to it. 

We will add some links for answers that may be open to opnions so we can evolve our understand of more scenarios possibles and try to learn for that. 

Thanks for participating on our prorject. 


# Questions from Developers / General Coding Patterns

As general principle of the code you see on this repo, we share the principle to have a good functional code that achieve the desired behavior, and want to be flexible to several developers styles. 

So, if you see something that could be improved or discussed let us know by:  [Starting a discussion](https://github.com/kubefirst/kubefirst/discussions/new?category=q-a) **or** [creating an Issue](https://github.com/kubefirst/kubefirst/issues/new?labels=enhancement,community%20wishlist&title=Feedback)


---
## Golang 

Today, `kubefirst` uses `golang` on most of its components/cli implementation of utilities. 

These are some of the patterns we use: 

### Cobra-cli 

We use [cobra-cli](https://github.com/spf13/cobra) to create our commands, it has its pro/cons.

If you would like to express some opnions on this, we have [this discussion](https://github.com/kubefirst/kubefirst/discussions/531) for it. 

### How to create a new command? 


This line will add a command under `actionCmd` to create a new `action`. `actionCmd` is special command to be parent of general commands that execute parts of installation or some developers use to test behaviors before creating a function for something. It a nice place to start as a sandbox. 

```bash 
cobra-cli add myCustomCommand -p 'actionCmd'
```

Please, use the CLI to create new commands, we know you can create it manually but we would like to keep the pre-generated style and structure. 

**Tip:** To install it, just run `go install github.com/spf13/cobra-cli@latest` 

### How a command looks like? 

We have as current practice this shape: 
```golang
var myActionCmd = &cobra.Command{
	Use:   "action-with-dash",
	Short: "...",
	Long: `...`,

	RunE: func(cmd *cobra.Command, args []string) error {
		includeMetaphorApps, err := cmd.Flags().GetBool("include-metaphor")
		if err != nil {
			return err
		}

        ...
		return nil
	},
}
```

Key points: 
- And command must return and `error` when it fails, so we can exit nicely from and execution that has a single command or a chain commands like [create](https://github.com/kubefirst/kubefirst/blob/main/cmd/create.go)
- Please, handle errors, and when it is part of the logic in execution and you need to fail the execution send the `error` on the return instead of direct `exit` or `panic`. 
- Ensure your command is using this signature: `RunE: func(cmd *cobra.Command, args []string) error` - in particular `RunE`. 

> We know, there is panic in the code, we are working to remove and improve error handling to all to be handled as described above. 
> 
> We know, there we call commands by `createGithubCmd.RunE(cmd, args)` instead of calling `Execute` when chaining commands. We may improve that later, but for today that produces the desired behavior we search from `cobra` tooling. We just want an easy way to have some functions that are also commands with flags. 

## Terraform

### Where do we use terraform?
[On our gitops template](https://github.com/kubefirst/gitops-template/tree/main/terraform)

### Where you can learn more?
[On our docs](https://docs.kubefirst.com/tooling/terraform.html)

### Do I need to install terraform?

Not really, installer install all the tools needed to execute all installation steps. Some developers have local installs for their tests. 

But you would have one already to use at you `$HOME/.k1/tools/terraform` if you executed `kubefirst init` once. 

That version will be the version we use for installation steps. 

## Argo/ArgoCD/Argo Workflows


## Vault


## Atlantis


## Others Tools

## Templates

### Gitops Templates

### Metaphor Templates


### CWFT


### K8S
