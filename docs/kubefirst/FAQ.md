# Frequently Asked Questions

## I ran into an issue, what should I do?

If an error occurs, try to run the command again as there is a `~/.kubefirst` file on your localhost that keeps track of your execution state. If it still doesn't work, each command create a log file in the `logs` folder, which is added at the root of the current location: the full path is displayed at the beginning of command outputs. This log file will often describe the details of what happened at the point of failure, and can help indicate the right steps for resolution.

If it's still not working, you can join our [Slack community](https://kubefirst.io/slack) and ask for help in the `#helping-hands` channel. You can also open an [issue](https://github.com/kubefirst/kubefirst/issues) describing the problems you are having. We'll gladly work through it with you.

## How do I tear it all down after I'm done checking it out?

Please find that information in the destroying your Kubefirst platform documentation related to your installation type:

- [GitHub](github/destroy.md)
- [GitLab](gitlab/destroy.md)
- [Local](local/destroy.md)

## Kubefirst Local is taking a long time to install, what should I do?

Kubefirst has a known bug downloading Terraform providers. This is a [known issue](https://github.com/kubefirst/kubefirst/issues/1055) and we are working on a fix. In the meantime, you can work around this by running the destroy command and trying it again.

## I'm getting an error about the `kubefirst` command not being found

Kubefirst wasn't correctly installed on your system. Please follow the [installation instructions](./local/install.md) again.

## Where can I have the services passwords?

The passwords are stored in the `~/.kubefirst` file. You can find the password for each service in the `services` section. The handoff screen (the purple screen at the end of the installation) also displays the passwords.

## Are there logs I can look at?

Yes, each command create a log file in the `logs` folder, which is added at the root of the current location: the full path is displayed at the beginning of command outputs. This log file will often describe the details of what happened at the point of failure, and can help indicate the right steps for resolution.
