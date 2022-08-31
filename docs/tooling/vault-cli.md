# Hashicorp Vault CLI

`vault` is a cli that we use to interact with vault from the command line.

### Releases

Vault keeps their client binaries available on their releases page:   
[https://github.com/hashicorp/vault/releases](https://github.com/hashicorp/vault/releases)

### Installation Instructions
**Warning: the following install is pinned, see [releases page](https://github.com/hashicorp/vault/releases) for a newer version.**
```bash
VERSION=1.8.0; curl -LO https://releases.hashicorp.com/vault/${VERSION}/vault_${VERSION}_darwin_amd64.zip
unzip vault_${VERSION}_darwin_amd64.zip
mv ./vault /usr/local/bin/vault
```
details: [https://www.vaultproject.io/downloads.html](https://www.vaultproject.io/downloads.html)

### Checking Your Vault Version

```bash
vault --version
```
expected result: `Vault v1.8.0 (82a99f14eb6133f99a975e653d4dac21c17505c7)`

### Configuration

You will need to get credentials to authenticate with vault before you can use the vault cli. 
Speak with your security administrator to get these credentials.

After executing your `vault login` command, you will be able to interact with the secrets that your ACL permits.

### Commands

[https://www.vaultproject.io/docs/commands](https://www.vaultproject.io/docs/commands)
