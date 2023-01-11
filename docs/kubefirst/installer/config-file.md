# Does kubefirst support config files? 

yes, here a sample of configs you can use. 


## Sample of AWS installation Gitlab

`./values.yaml`: 
```yaml 
config:
  admin-email:  user@domain.com
  cloud: aws
  hosted-zone-name: my.domain.com
  profile: default
  bot-password: myAmazingPassword
  cluster-name: mycluster
```

```bash 
kubefirst init  -c ./values.yaml
```

## Sample of AWS installation Github

`./values.yaml`: 
```yaml 
config:
  admin-email:  user@domain.com
  cloud: aws
  hosted-zone-name: my.domain.com
  profile: default
  bot-password: myAmazingPassword
  cluster-name: mycluster
  github-user: my_github_user
  github-org: my_github_org
```

```bash 
kubefirst init  -c ./values.yaml
```

## Sample of Local installation

`./kubefirst-config.yaml`: 
```yaml 
config:
  log-level: debug
```

```bash 
kubefirst local
```
If you create a file as described below and run the the command `kubefirst local`, the CLI will read and load all the values and overwrite the default ones. 