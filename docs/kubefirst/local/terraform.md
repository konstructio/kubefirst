# Terraform and Atlantis
`terraform` is our infrastructure as code layer and we manage our terraform workflows with `atlantis` automation.

## Making Changes In Terraform

### Automatic Plans With Atlantis
Any merge request that includes a .tf file will prompt `atlantis` to wake up and run your terraform plan. Atlantis will post the plan's result to your merge request as a comment within a minute or so.

Review and eventually approve the merge request.

### Apply and Merge
Add the comment `atlantis apply` in the approved merge request. This will prompt atlantis to wake up and run your `terraform apply`.

The apply results will be added to your pull request comments by atlantis.

If the apply is successful, your code will automatically be merged with master, your merge request will be closed, and the state lock will be removed in atlantis.

## Managing Terraform State
Your terraform state is stored in an s3 bucket named `k1-state-store-xxxxxx`.

The s3 bucket implements versioning, so if your terraform state store ever gets corrupted, you can roll it back to a previous state without too much trouble.

Note that terraform at times needs to store secrets in your state store, and therefore access to this s3 bucket should be restricted to only the administrators who need it.
