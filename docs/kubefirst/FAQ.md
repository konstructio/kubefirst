# Frequently Asked Questions

## I ran into an issue, what should I do?

If an error occurs, try to run the command again as there is a `~/.kubefirst` file on your localhost that keeps track of your execution state. If it still doesn't work, each command create a log file in the `logs` folder, which is added at the root of the current location: the full path is displayed at the beginning of command outputs. This log file will often describe the details of what happened at the point of failure, and can help indicate the right steps for resolution.

If it's still not working, you can join our [Slack community](https://kubefirst.io/slack) and ask for help in the `#helping-hands` channel. You can also open an [issue](https://github.com/kubefirst/kubefirst/issues) describing the problems you are having. We'll gladly work through it with you.

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1184)_</sub>

## How do I tear it all down after I'm done checking it out?

Please find that information in the destroying your Kubefirst platform documentation related to your installation type:

- [GitHub](github/destroy.md)
- [GitLab](gitlab/destroy.md)
- [Local](local/destroy.md)

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1186)_</sub>

## Kubefirst Local is taking a long time to install, what should I do?

Kubefirst has a known bug downloading Terraform providers. This is a [known issue](https://github.com/kubefirst/kubefirst/issues/1055) and we are working on a fix. In the meantime, you can work around this by running the destroy command and trying it again.

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1187)_</sub>

## I'm getting an error about the `kubefirst` command not being found

Kubefirst wasn't correctly installed on your system. Please follow the [installation instructions](./local/install.md) again.

## I'm stuck with artifacts after a failed local installation and can't continue

If you still cannot complete the installation due to artifacts after completing a local destroy, you may have to reset your state manually.

## Manual Teardown

If the above command fails to complete due to unforeseen circumstances, you can then manually delete the git repositories named:

- gitops
- metaphor-frontend (only exists if you complete Kubefirst local provisioning)

You can then manually delete the k3d cluster with the command `k3d cluster delete kubefirst` or `~/.k1/tools/k3d cluster delete kubefirst` if you don't have k3d installed.

To delete your github assets that we created, log into your personal github and remove the following:

- gitops repo
- metaphor-frontend repo

Manual Destroy CLI Example:
```
gh repo delete <GITHUB_USERNAME>/metaphor-go --confirm
gh repo delete <GITHUB_USERNAME>/metaphor-frontend --confirm
gh repo delete <GITHUB_USERNAME>/metaphor --confirm
gh repo delete <GITHUB_USERNAME>/gitops --confirm

---
$HOME/.k1/tools/k3d cluster delete kubefirst
---
kubefirst clean
```

## I'm experiencing timeouts when Kubefirst deploys ArgoCD / Vault through helm installation

You may need a more stable connection / higher download speed. Check with your internet provider or use an online speed test to confirm you have at least 100mbps download speed, or else you may experience timeouts.

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1188)_</sub>

## Where can I have the services passwords?

The passwords are stored in the `~/.kubefirst` file. You can find the password for each service in the `services` section. The handoff screen (the purple screen at the end of the installation) also displays the passwords.

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1189)_</sub>

## Are there logs I can look at?

Yes, each command create a log file in the `logs` folder, which is added at the root of the current location: the full path is displayed at the beginning of command outputs. This log file will often describe the details of what happened at the point of failure, and can help indicate the right steps for resolution.

<sub>_[Do you want help to improve this answer?](https://github.com/kubefirst/kubefirst/discussions/1190)_</sub>
