# Run this ONCE before everything else to create the remote state bucket.
# Usage:
#   cd Infra/bootstrap
#   terraform init
#   terraform apply
#
# After apply, copy the bucket name into Infra/backend.tf, then:
#   cd ../
#   terraform init   (will migrate local state to S3 automatically)

terraform {
  required_version = ">= 1.10.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.30"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

data "aws_caller_identity" "current" {}

locals {
  bucket_name = "ecommerce-tfstate-${data.aws_caller_identity.current.account_id}"
}

resource "aws_kms_key" "tfstate" {
  description             = "KMS key for Terraform state encryption"
  deletion_window_in_days = 10
  enable_key_rotation     = true

  tags = { Name = "terraform-state-key", ManagedBy = "terraform-bootstrap" }
}

resource "aws_kms_alias" "tfstate" {
  name          = "alias/terraform-state"
  target_key_id = aws_kms_key.tfstate.id
}

resource "aws_s3_bucket" "tfstate" {
  bucket        = local.bucket_name
  force_destroy = false

  tags = { Name = local.bucket_name, ManagedBy = "terraform-bootstrap" }
}

resource "aws_s3_bucket_versioning" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.tfstate.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "tfstate" {
  bucket                  = aws_s3_bucket.tfstate.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# use_lockfile (TF >= 1.10) stores a .tflock file using S3 conditional writes.
# No DynamoDB table needed — S3 If-None-Match provides the same atomic lock guarantee.

output "bucket_name" {
  value = aws_s3_bucket.tfstate.bucket
}

output "kms_key_arn" {
  value = aws_kms_key.tfstate.arn
}
