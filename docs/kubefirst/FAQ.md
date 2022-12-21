# Frequently Asked Questions

## I ran into an issue, what should I do?

If an error occurs, try to run the command again as there is a `~/.kubefirst` file on your localhost that keeps track of your execution state. If it still doesn't work, each command create a log file in the `logs` folder, which is added at the root of the current location: the full path is displayed at the beginning of command outputs. This log file will often describe the details of what happened at the point of failure, and can help indicate the right steps for resolution.

If it's still not working, you can join our [Slack community](https://kubefirst.io/slack) and ask for help in the `#helping-hands` channel. You can also open an [issue](https://github.com/kubefirst/kubefirst/issues) describing the problems you are having. We'll gladly work through it with you.

## How do I tear it all down after I'm done checking it out?

Please find that information in the destroying your Kubefirst platform documentation related to your installation type:

- [GitHub](github/destroy.md)
- [GitLab](gitlab/destroy.md)
- [Local](local/destroy.md)
