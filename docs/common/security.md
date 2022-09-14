# Security

Kubefirst work on top of cloud providers, and your cloud credentials are used to previsioning the Kubefirst platform. We
have a tremendous respect when it comes to your Cloud provider data, and we take security very seriously.

We use different approaches to close any possible attack surface using different technologies and strategies. On the 
service side, we have Vault to store and encrypt your sensitive data. Every resource that is exposed to the outside 
world is encrypt using SSL/TLS via Let's Encrypt.

## Assume Role
Kubefirst also provides an approach to run the previsioning process on less privileges strategy via 
[AWS Assume Role](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html). The idea is to basically 
configure a AWS role with the permissions you would like to provide to the Kubefirst installer, and provide that role to
the [Assume Role Kubefirst command](../tooling/kubefirst-cli.md) via de `init` command.
