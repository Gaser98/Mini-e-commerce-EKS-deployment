# GitHub Actions authenticates to AWS via OIDC federation — no stored access keys anywhere.
#
# Flow:
#   1. GitHub generates a short-lived JWT (the OIDC token) for the workflow run
#   2. The workflow calls STS AssumeRoleWithWebIdentity, passing that JWT
#   3. STS validates the JWT against this OIDC provider
#   4. STS returns temporary credentials scoped to github_actions role (15 min TTL)
#
# The repo constraint in the Condition block ensures only YOUR repo can assume this role.

resource "aws_iam_openid_connect_provider" "github" {
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]

  # GitHub's OIDC thumbprints — update if GitHub rotates their cert
  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd",
  ]
}

resource "aws_iam_role" "github_actions" {
  name = "github-actions-ecommerce"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringLike = {
          # Allows all branches and PR refs in this repo
          "token.actions.githubusercontent.com:sub" = "repo:Gaser98/db-design-project:*"
        }
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "github_actions" {
  name = "github-actions-policy"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      # ECR login token (account-level, no specific resource)
      {
        Effect   = "Allow"
        Action   = ["ecr:GetAuthorizationToken"]
        Resource = "*"
      },
      # Push images to the API repo only
      {
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
          "ecr:BatchGetImage",
          "ecr:GetDownloadUrlForLayer",
        ]
        Resource = aws_ecr_repository.api.arn
      },
      # Read cluster details to generate kubeconfig
      {
        Effect   = "Allow"
        Action   = ["eks:DescribeCluster"]
        Resource = module.eks.cluster_arn
      },
      # Read/write Terraform remote state
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket", "s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = [
          "arn:aws:s3:::ecommerce-tfstate-${data.aws_caller_identity.current.account_id}",
          "arn:aws:s3:::ecommerce-tfstate-${data.aws_caller_identity.current.account_id}/*",
        ]
      },
      # Read secrets for terraform data sources
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
        Resource = "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:ecommerce/*"
      },
      # Read-only EC2 for terraform plan/apply (VPC data sources)
      {
        Effect   = "Allow"
        Action   = ["ec2:Describe*"]
        Resource = "*"
      },
    ]
  })
}

output "github_actions_role_arn" {
  description = "Paste this into the GitHub Actions workflow role-to-assume"
  value       = aws_iam_role.github_actions.arn
}
