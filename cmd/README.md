# Overview

General overview of the code, to help shuffling parts around.

# Globals Variables

| Variable|Type | Use |
|:--|:--|:--|
|cleanCmd|Command| Clean command|
|createCmd|Command| Create command|
|destroyCmd|Command| Destroy command|
|nebulousCmd|Command| CLI main command |
|versionCmd|Command| Version command|
|rootCmd|Command| Root command command|
|infoCmd|Command| Info command|
|initCmd|Command| Init command - pre-provision command|
|home|String|User Home dir or Work dir|
|kubectlClientPath|String|Kubectl CLI path|
|kubeconfigPath|String|Kubeconfig Path|
|localOs|String|Host OS|
|localArchitecture|String|Host OS architecture|
|terraformPath|String|Terraform CLI path|
|helmClientPath|String|Helm CLI path|
|dryrunMode|Bool|If installer should run in dry-run mode|
|Trackers|Trecker|Map of trackers|
|vaultRootToken|String|Root token for vault|
|gitlabToolboxPodName|String|Toolbox pod name|
|gitlabSecretClient|coreV1Types.SecretInterface|Client shorthand to interface|
|vaultSecretClient|coreV1Types.SecretInterface|Client shorthand to interface|
|argocdSecretClient|coreV1Types.SecretInterface|Client shorthand to interface|
|gitlabSecretClient|coreV1Types.PodInterface|Client shorthand to interface|
|cfgFile|String| .flare config file|
|NebolousVersion|String|CLI version|

