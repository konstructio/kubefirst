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
  cloud: aws
  git-provider: gitlab

```

```bash 
kubefirst init  -c ./values.yaml
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
  cloud: aws
```

```bash 
kubefirst init  -c ./values.yaml
```

