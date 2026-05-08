locals {
  env           = terraform.workspace
  is_production = local.env == "production"
  is_staging    = local.env == "staging"
  is_ephemeral  = startswith(local.env, "pr-")

  common_tags = {
    Project     = "ecommerce"
    Environment = local.env
    ManagedBy   = "terraform"
  }
}
