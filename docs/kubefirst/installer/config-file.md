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

## Sample of Local Mode - Github

`./values.yaml`: 
```yaml
config:
  admin-email:  user@domain.com
  cloud: k3d
  cluster-name: mycluster
  github-user: my_github_user
  github-org: my_github_org
```

```bash 
kubefirst init  -c ./values.yaml
```
