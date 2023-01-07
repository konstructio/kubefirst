# Roadmap

Here's what we plan to release for Kubefirst v1.11.

Please note that we may not be able to prioritize all requests, but if you want us to fix a specific issue not listed in this roadmap, please add a comment on it or create one if it wasn't reported yet. Same goes for feature requests. For anything else, please ask in #helping-hands in our [Slack community](http://kubefirst.io/slack).

## Kubefirst v1.11

### New Features

- Introduce a publicly available intro chart and image for metaphor [#752](https://github.com/kubefirst/kubefirst/issues/752)
- Introduce log levels [#756](https://github.com/kubefirst/kubefirst/issues/756)
- Introduce metaphor slim [#749](https://github.com/kubefirst/kubefirst/issues/749)

#### Kubefirst Local

- Add `kubefirst local destroy` command for destroying local [#864](https://github.com/kubefirst/kubefirst/issues/864)
- Add local Ingress to metaphor slim [#879](https://github.com/kubefirst/kubefirst/issues/879)
- DNS support: provide a local dns similar to the cloud dns experience [#747](https://github.com/kubefirst/kubefirst/issues/747)
- Ingress support: implement Ingress controller for Kubefirst local [#745](https://github.com/kubefirst/kubefirst/issues/745)
- SSL support: enable https access to local services throughout kubefirst local [#746](https://github.com/kubefirst/kubefirst/issues/746)

### Improvements

- Add retry logic to create ngrok tunnel [#929](https://github.com/kubefirst/kubefirst/pull/929)
- Ask user for confirmation at github token screen [#863](https://github.com/kubefirst/kubefirst/issues/863)
- Establish a frictionless user password management through Vault [#748](https://github.com/kubefirst/kubefirst/issues/748)
- Improve template download logic [#757](https://github.com/kubefirst/kubefirst/issues/757)
- kubefirst console - HashiCorp Vault should be in tile position #2 on aws github, aws gitlab, and k3d github stacks [#824](https://github.com/kubefirst/kubefirst/issues/824)
- Kubefirst EKS needs upgrade to newest supported version [#811](https://github.com/kubefirst/kubefirst/issues/811)

#### Kubefirst Console

- Provide telemetry on console activity [#754](https://github.com/kubefirst/kubefirst/issues/754)

#### Kubefirst Local

- Improve destroy for local [#794](https://github.com/kubefirst/kubefirst/issues/794)

### Fixes

- `restoreSSL` fails to backup certificates due to a missing bucket [#915](https://github.com/kubefirst/kubefirst/issues/915)
- 1.11 - cloud - destroy not working [#901](https://github.com/kubefirst/kubefirst/issues/901)
- AWS - Github Terraform not following install Region [#853](https://github.com/kubefirst/kubefirst/issues/853)
- AWS_PROFILE isn't set when executing kubectl and causes authentication errors [#838](https://github.com/kubefirst/kubefirst/issues/838)
- Cloud install - kubeconfig should not be added to the gitops repository [#926](https://github.com/kubefirst/kubefirst/issues/926)
- ErrImagePull on actions-runner-metaphor-frontend pod [#924](https://github.com/kubefirst/kubefirst/issues/924)
- Fix backupSSL not working on 1.11 (regression) [#900](https://github.com/kubefirst/kubefirst/issues/900)
- Github Runners is being deployed in Gitlab flavor [#904](https://github.com/kubefirst/kubefirst/issues/904)
- Include env var for region [#944](https://github.com/kubefirst/kubefirst/pull/944)
- Ngrok authentication failed [#850](https://github.com/kubefirst/kubefirst/issues/850)
- Progress bars are broken for installations [#919](https://github.com/kubefirst/kubefirst/issues/919)
- Removed VOUCH_DOCKER_REGISTRY/VOUCH_DOCKER_TAG rules for deton (regression) [#897](https://github.com/kubefirst/kubefirst/issues/897)

#### Kubefirst Console

- CrashLoopBackOff on Kubefirst Console pod [#923](https://github.com/kubefirst/kubefirst/issues/923)
- Kubefirst Console not working on 1.11 [#899](https://github.com/kubefirst/kubefirst/issues/899)

#### Kubefirst GitHub

- Metaphor - Publish/Build is not working for GitHub [#925](https://github.com/kubefirst/kubefirst/issues/925)

#### Kubefirst Local

- `kubefirst local` destroy fails when the `~/.k1` folder isn't empty [#930](https://github.com/kubefirst/kubefirst/issues/930)
- `kubefirst local` fails without an error message to the user [#912](https://github.com/kubefirst/kubefirst/issues/912)
- `kubefirst local` installation produces invalid certificate errors on all hostnames [#910](https://github.com/kubefirst/kubefirst/issues/920)
- local - when using branch is producing a main-less repo [#818](https://github.com/kubefirst/kubefirst/issues/818)
- Local Setup - Release branch issue with Atlantis [#817](https://github.com/kubefirst/kubefirst/issues/817)
- Welcome message prints long before local installation is complete [#918](https://github.com/kubefirst/kubefirst/issues/918)
