# CWFT 

## What are CWFT? 

They are [Argo Workflow Template](https://argoproj.github.io/argo-workflows/cluster-workflow-templates/) used to create [Argo Workflows](https://argoproj.github.io/argo-workflows/workflow-concepts/#the-workflow). 


## How they are used on Kubefirst? 

Kubefirst use **CWFT** as building blocks of CI pipelines that used on both of our git providers(github and gitlab). As part of kubefirst we provide a set of demo applications called mataphor, that uses our **CWFTs** tol help users to be inspired to create their onws based on the ones provided and update the existin ones. 


## Are they shared between multiple repos? 

Yes, the idea on kubefirst is that based on lessons learned on the gitops journey we want to support the reuse of the CWFTs, so teams can share practices. 


## How can you create yours? 

**CWFT** are a special case of [Argo WorkflowTemplate](https://argoproj.github.io/argo-workflows/fields/#workflowtemplate). The only difference they are CRDs on cluster wide visibility. 

[Learn more](https://argoproj.github.io/argo-workflows/cluster-workflow-templates/)

