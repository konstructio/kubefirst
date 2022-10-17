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
```

```bash 
kubefirst init  -c ./values.yaml
```

