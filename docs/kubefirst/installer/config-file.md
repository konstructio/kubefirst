# Does kubefirst support config files? 

yes, here a sample of configs you can use. 


## Sample of AWS installation Gitlab

`./values.yaml`: 
```yaml 
config:
  admin-email:  user@domain.com
  hosted-zone-name: my.domain.com
  profile: default
  bot-password: myAmazingPassword
  cluster-name: mycluster
```

```bash 
kubefirst init  -c ./values.yaml --cloud aws --git-provider gitlab
```

## Sample of AWS installation Github

`./values.yaml`: 
```yaml 
config:
  admin-email:  user@domain.com
  hosted-zone-name: my.domain.com
  profile: default
  bot-password: myAmazingPassword
  cluster-name: mycluster
  github-owner: my_github_org
```

```bash 
kubefirst init  -c ./values.yaml  --cloud aws 
```

# Notes

- The flag `--cloud` is not supported via config file
- The flag `--git-provider` is not supported via config file, if not passed it will be assumed as a `github` installation.
- `kubefirst local` has not formal support to config files
