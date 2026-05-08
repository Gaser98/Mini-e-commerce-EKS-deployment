variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "azs" {
  description = "Availability zones"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
  default     = "demo"
}

variable "db_username" {
  description = "PostgreSQL username"
  type        = string
  default     = "demo"
}

# db_password removed — credentials are now read from AWS Secrets Manager in data.tf.
# Bootstrap: aws secretsmanager create-secret --name "ecommerce/<env>/rds" \
#   --secret-string '{"username":"demo","password":"<pass>","connection_string":"postgres://..."}'
