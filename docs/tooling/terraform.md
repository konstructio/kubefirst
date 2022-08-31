# Terraform

`terraform` is the cli for directly negotiating infrastructure as code when not using Atlantis for automated operations. This may include administrative tasks such as terraform state mv or import operationss.

### Releases

Terraform keeps their client binaries available on their releases page:   
[https://github.com/hashicorp/terraform/releases](https://github.com/hashicorp/terraform/releases)

## Installation Instructions
### `tfenv` & `terraform`

We use `tfenv` to be able to easily pivot between terraform versions.

### `tfenv` Installation Instructions

```
brew install tfenv
```
details:
[https://github.com/tfutils/tfenv](https://github.com/tfutils/tfenv)

### Installing Terraform With `tfenv`

```
tfenv install 1.0.3
```

### Checking Your Terraform Version

```bash
terraform version
```
expected result: `Terraform v1.0.3`

### Additional Notes

You're not required to use tfenv to manage your terraform versions and installations. To install terraform directly see [https://www.terraform.io/downloads.html](https://www.terraform.io/downloads.html)
