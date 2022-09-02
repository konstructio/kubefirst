# AWS CLI

`aws` is the cli that we use to interact with aws cloud from the command line

### Releases

AWS keeps their CLI versions on their releases page:   
[https://github.com/aws/aws-cli/releases](https://github.com/aws/aws-cli/releases)

### Installation Instructions
**Warning: the following install is pinned, see [releases page](https://github.com/aws/aws-cli/releases) for a newer version.**
```bash
VERSION=2.2.25; curl "https://awscli.amazonaws.com/AWSCLIV2-${VERSION}.pkg" -o "AWSCLIV2.pkg"
sudo installer -pkg AWSCLIV2.pkg -target /
```
details: [https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html)

### Checking Your AWS CLI Version

```bash
aws --version
```
expected result: `aws-cli/2.2.25 Python/3.8.8 Darwin/18.7.0 exe/x86_64 prompt/off`
