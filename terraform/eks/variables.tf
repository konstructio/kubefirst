variable "region" {
  default = "us-east-1"
}

variable "map_accounts" {
  description = "Additional AWS account numbers to add to the aws-auth configmap."
  type        = list(string)

  default = []
}
variable "map_roles" {
  description = "Additional IAM roles to add to the aws-auth configmap."
  type = list(object({
    rolearn  = string
    username = string
    groups   = list(string)
  }))

  default = [ # todo need to remove this role
    {
      rolearn  = "arn:aws:iam::659548672500:role/KubernetesAdmin"
      username = "admin"
      groups   = ["system:masters"]
    },
  ]
}

variable "map_users" {
  description = "Additional IAM users to add to the aws-auth configmap."
  type = list(object({
    userarn  = string
    username = string
    groups   = list(string)
  }))

  # todo need to pass this user arn in from script execution
  default = []
}

variable "k8s_admin" {
  type    = string
  default = "arn:aws:iam::aws:policy/AdministratorAccess"
}

variable "k8s_worker_node_policy_arns" {
  type = list(string)
  # todo note - went from ECR ReadOnly to PowerUser -- default[0] and default[1] are REQUIRED by EKS, even though PowerUser should trump ReadOnly...
  default = ["arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy", "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly", "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser", "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy", "arn:aws:iam::aws:policy/AdministratorAccess"]
}

variable "cluster_name" {
  type = string
}

variable "aws_account_id" {
  type = string
}
variable "iam_user_arn" {
  type = string
}