# data sources read existing infrastructure — Terraform observes but does not manage them.
# Use these instead of hardcoding account IDs, ARNs, or passing secrets as variables.

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# Pull RDS credentials from Secrets Manager.
# The secret must be created manually (bootstrap) before the first terraform apply:
#   aws secretsmanager create-secret \
#     --name "ecommerce/production/rds" \
#     --secret-string '{"username":"demo","password":"<strong-password>","connection_string":"postgres://demo:<pass>@<host>:5432/demo?sslmode=require"}'
data "aws_secretsmanager_secret_version" "db_creds" {
  secret_id = "ecommerce/${local.env}/rds"
}

data "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id = "ecommerce/${local.env}/jwt"
}

# Read the ECR repo created in ecr.tf (used in github-oidc.tf policy)
data "aws_ecr_repository" "api" {
  name       = aws_ecr_repository.api.name
  depends_on = [aws_ecr_repository.api]
}

# Read the EKS OIDC TLS certificate — used to verify tokens from pods (IRSA)
data "tls_certificate" "eks" {
  url = module.eks.cluster_oidc_issuer_url
}
