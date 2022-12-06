# Frequently Asked Questions

## I ran into an issue, what should I try?

If an error occurrs during install or uninstall, you can often just run the command again. There's a `~/.kubefirst` file on your localhost that keeps track of your execution state. If rerunning doesn't work, move on to the next step.

## I ran into an issue and rerunning didin't help, what should I try now?

There's a log file that gets printed right after the `kubefirst local` command starts. This log file will often destribe the details of what happened at the point of failure, and can help indicate the right steps for resolution.

## I checked the logs and still can't figure out what to do next - what now?

Open an issue against the kubefirst/kubefirst repo on github. We'll gladly work through it with you.
[https://github.com/kubefirst/kubefirst/issues](https://github.com/kubefirst/kubefirst/issues)

## I hate keeping up on GitHub issues - can I just talk to you on Slack?

We'd love that! Join the `kubefirst-community` Slack workspace by following [this link].(https://join.slack.com/t/kubefirst-community/shared_invite/zt-r0r9cfts-OVnH0ooELDLm9n9p2aU7fw)

## How do I tear down kubefirst local when I'm done checking it out?

See [./destroy.md](./destroy.md)

## What if destroy doesn't work?

To delete the local k3d cluster run

```
k3d cluster delete kubefirst
```

To delete your github assets that we created, log into your personal github and remove the following:
- gitops repo
- metaphor repo
- metaphor-go repo
- metaphor-frontend repo
- developers team
- admins team
