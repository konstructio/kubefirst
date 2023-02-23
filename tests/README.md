# End to End tests

This directory contains end to end tests to be run against a running Kubefirst instance of the [local](https://docs.kubefirst.io/kubefirst/local/install.html), [AWS with GitHub](https://docs.kubefirst.io/kubefirst/github/install.html), and [AWS with GitLab](https://docs.kubefirst.io/kubefirst/gitlab/install.html) installations. After a successful installation, the tests can be run to verify that the:

- installation was successful
- cluster is working as expected
- downloaded repositories are working as expected
- Metaphor application is working as expected
- Traefik ingress controller rules are working as expected
- TLS certificates are working as expected
- Vault is working as expected
- Vault initial token is able to be used to login to Vault
- Kubefirst process is able to create new GitHub users via IAC/Webhooks/Atlantis/Terraform
- newly created users are able to login into Vault

## Taskfile to trigger sequential execution of the tests

Kubefirst make use of [Taskfile](https://github.com/go-task/task) (instead of makefile), to trigger sequential execution of the tests. The [Taskfile](../Taskfile.yaml) is located in the root of this repository. The following test cases are avaialble:

### test Traefik rules and TLS certificates

```bash
task integration-test-for-tls-localdev:
```

### test Metaphor application

```bash
task e2e-test-local-metaphors:
```

### test user creation via IAC/Webhooks/Atlantis/Terraform, and check if is able to login with the new user

```bash
task e2e-test-github-user-creation-and-login:
```
