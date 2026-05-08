# Remote state backend using S3 + native locking (Terraform >= 1.10).
#
# use_lockfile = true replaces DynamoDB. When Terraform runs, it creates a
# <key>.tflock file in S3 using an If-None-Match conditional write. If another
# process already holds the lock, S3 rejects the write — same guarantee as DynamoDB,
# zero extra infrastructure.
#
# Workspace isolation: each workspace gets its own state file automatically.
#   default workspace  → ecommerce/terraform.tfstate
#   staging workspace  → env:/staging/ecommerce/terraform.tfstate
#   pr-42 workspace    → env:/pr-42/ecommerce/terraform.tfstate
#
# Migration from local state:
#   1. Run Infra/bootstrap first to create the bucket
#   2. Add this file
#   3. Run: terraform init
#      Terraform detects the new backend and asks "copy existing state? yes"
#   4. Delete Infra/terraform.tfstate and Infra/terraform.tfstate.backup
#   5. Add both files to .gitignore

terraform {
  backend "s3" {
    bucket       = "ecommerce-tfstate-776235864987"
    key          = "ecommerce/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    kms_key_id   = "alias/terraform-state"
    use_lockfile = true
  }
}
