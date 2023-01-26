# End to End tests (for local / cloud(aws GitHub/aws GitLab) is work in progress)

### This directory contains end to end tests to be run against a running Kubefirst instance. After a successful installation, the tests can be run to:

- verify that the installation was successful
- verify that the cluster is working as expected
- verify that downloaded repositories are working as expected
- verify that Metaphor application is working as expected
- verify that Traefik ingress controller rules are working as expected
- verify that TLS certificates are working as expected
- verify that Vault is working as expected
- verify that the Vault initial token is able to be used to login to Vault
- verify that Kubefirst is able to create new GitHub users via IAC/Webhooks/Atlantis/Terraform
- verify that the newly created users are able to login into Vault

### Taskfile to trigger sequential execution of the tests

Kubefirst make use of Taskfile (instead of makefile), to trigger sequential execution of the tests. The Taskfile is located in the root of the repository. The following test cases are avaialble:

**test Traefik rules and TLS certificates**

```bash
task integration-test-for-tls-localdev:
```

**test Metaphor application**

```bash
task e2e-test-local-metaphors:
```

**test user creation via IAC/Webhooks/Atlantis/Terraform, and check if is able to login with the new user**

```bash
task e2e-test-github-user-creation-and-login:
```
