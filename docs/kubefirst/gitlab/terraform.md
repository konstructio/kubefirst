# Terraform and Atlantis
`terraform` is our infrastructure as code layer and we manage our Terraform workflows with `atlantis` automation.

## Making Changes In Terraform

### Automatic Plans With Atlantis
Any merge request that includes a .tf file will prompt `atlantis` to wake up and run your Terraform plan. Atlantis will post the plan's result to your merge request as a comment within a minute or so.

Review and eventually approve the merge request.

### Apply and Merge
Add the comment `atlantis apply` in the approved merge request. This will prompt Atlantis to wake up and run your `terraform apply`.

The apply results will be added to your pull request comments by Atlantis.

If the apply is successful, your code will automatically be merged with master, your merge request will be closed, and the state lock will be removed in Atlantis.

## Managing Terraform State
Your Terraform state is stored in an S3 bucket named `k1-state-store-xxxxxx`.

The S3 bucket implements versioning, so if your Terraform state store ever gets corrupted, you can roll it back to a previous state without too much trouble.

Note that Terraform at times needs to store secrets in your state store, and therefore access to this S3 bucket should be restricted to only the administrators who need it.
