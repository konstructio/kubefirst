terraform {
  required_version = ">= 0.12.0"
}

provider "random" {
  version = "~> 2.1"
}

provider "local" {
  version = "~> 1.2"
}

provider "null" {
  version = "~> 2.1"
}

provider "template" {
  version = "~> 2.1"
}

data "aws_eks_cluster" "cluster" {
  name = module.eks.cluster_id
}

data "aws_eks_cluster_auth" "cluster" {
  name = module.eks.cluster_id
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
  load_config_file       = false
  version                = "~> 1.11"
}

data "aws_availability_zones" "available" {
}

locals {
  cluster_name = var.cluster_name
}

resource "random_string" "suffix" {
  length  = 8
  special = false
}

resource "aws_security_group" "worker_group_mgmt_one" {
  name_prefix = "worker_group_mgmt_one"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8",
    ]
  }
}

resource "aws_security_group" "worker_group_mgmt_two" {
  name_prefix = "worker_group_mgmt_two"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      "192.168.0.0/16",
    ]
  }
}

resource "aws_security_group" "all_worker_mgmt" {
  name_prefix = "all_worker_management"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8",
      "172.16.0.0/12",
      "192.168.0.0/16",
    ]
  }
}
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "2.47.0"

  name                 = "kubefirst-vpc"
  cidr                 = "10.0.0.0/16"
  azs                  = data.aws_availability_zones.available.names
  private_subnets      = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets       = ["10.0.4.0/24", "10.0.5.0/24", "10.0.6.0/24"]
  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  public_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                      = "1"
  }

  private_subnet_tags = {
    "kubernetes.io/cluster/${local.cluster_name}" = "shared"
    "kubernetes.io/role/internal-elb"             = "1"
  }
}

module "eks" {
  version         = "17.20.0"
  source          = "terraform-aws-modules/eks/aws"
  cluster_name    = local.cluster_name
  cluster_version = "1.20"
  subnets         = module.vpc.private_subnets

  tags = {
    ClusterName = "kubefirst"
  }

  vpc_id = module.vpc.vpc_id

  map_roles = concat(var.map_roles, [{
    rolearn  = "arn:aws:iam::${var.aws_account_id}:role/KubernetesAdmin"
    username = "admin"
    groups   = ["system:masters"]
    }, {
    rolearn  = aws_iam_role.kubefirst_worker_nodes_role.arn
    username = "system:node:{{EC2PrivateDNSName}}"
    groups   = ["system:bootstrappers", "system:nodes"]
  }])
  map_users = concat(var.map_users, [{
    userarn  = var.iam_user_arn
    username = "admin"
    groups   = ["system:masters"]
  }])
  map_accounts = var.map_accounts
}


resource "aws_eks_node_group" "preprod_nodes" {
  cluster_name    = module.eks.cluster_id
  node_group_name = "preprod-nodes"
  node_role_arn   = aws_iam_role.kubefirst_worker_nodes_role.arn
  subnet_ids      = module.vpc.private_subnets
  ami_type        = "AL2_x86_64"
  disk_size       = 50

  labels = {
    "workload" = "preprod"
  }

  scaling_config {
    desired_size = 2
    max_size     = 2
    min_size     = 2
  }

  # Ensure that IAM Role permissions are created before and deleted after EKS Node Group handling.
  # Otherwise, EKS will not be able to properly delete EC2 Instances and Elastic Network Interfaces.
  depends_on = [
    module.eks
  ]
}

resource "aws_eks_node_group" "mgmt_nodes" {
  cluster_name    = module.eks.cluster_id
  node_group_name = "mgmt-nodes"
  node_role_arn   = aws_iam_role.kubefirst_worker_nodes_role.arn
  subnet_ids      = module.vpc.private_subnets
  ami_type        = "AL2_x86_64"
  disk_size       = 50

  labels = {
    "workload" = "mgmt"
  }

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  # Ensure that IAM Role permissions are created before and deleted after EKS Node Group handling.
  # Otherwise, EKS will not be able to properly delete EC2 Instances and Elastic Network Interfaces.
  depends_on = [
    module.eks
  ]
}
resource "aws_eks_node_group" "production_nodes" {
  cluster_name    = module.eks.cluster_id
  node_group_name = "production-nodes"
  node_role_arn   = aws_iam_role.kubefirst_worker_nodes_role.arn
  subnet_ids      = module.vpc.private_subnets
  ami_type        = "AL2_x86_64"
  disk_size       = 50

  labels = {
    "workload" = "production"
  }

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  # Ensure that IAM Role permissions are created before and deleted after EKS Node Group handling.
  # Otherwise, EKS will not be able to properly delete EC2 Instances and Elastic Network Interfaces.
  depends_on = [
    module.eks
  ]
}

resource "random_string" "random" {
  length  = 16
  special = false
}

resource "aws_iam_role" "kubefirst_worker_nodes_role" {
  name = "kubefirst-worker-nodes-role-${random_string.random.result}"

  assume_role_policy = <<EOT
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": { 
        "AWS": [
          "arn:aws:iam::${var.aws_account_id}:root"
        ]
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Sid": "EKSWorkerAssumeRole",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOT
}

resource "aws_iam_role_policy_attachment" "admin_policy_attach" {
  role       = aws_iam_role.kubefirst_worker_nodes_role.name
  policy_arn = var.k8s_admin
}

resource "aws_iam_role_policy_attachment" "worker_policy_attach" {
  count      = length(var.k8s_worker_node_policy_arns)
  role       = aws_iam_role.kubefirst_worker_nodes_role.name
  policy_arn = var.k8s_worker_node_policy_arns[count.index]
}
