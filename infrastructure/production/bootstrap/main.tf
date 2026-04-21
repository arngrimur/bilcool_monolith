terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Intentionally uses local state — this config creates the bucket that the
  # main config uses as its remote backend, so it cannot use that backend itself.
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project   = "bilcool"
      ManagedBy = "terraform"
    }
  }
}

variable "aws_region" {
  type    = string
  default = "eu-north-1"
}

# ── Terraform state bucket ────────────────────────────────────────────────────

resource "aws_s3_bucket" "terraform_state" {
  bucket = "bilcool-terraform-state"
}

resource "aws_s3_bucket_versioning" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "terraform_state" {
  bucket = aws_s3_bucket.terraform_state.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "terraform_state" {
  bucket                  = aws_s3_bucket.terraform_state.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ── Lambda artifacts bucket ───────────────────────────────────────────────────

resource "aws_s3_bucket" "lambda_artifacts" {
  bucket = "bilcool-lambda-artifacts"
}

resource "aws_s3_bucket_server_side_encryption_configuration" "lambda_artifacts" {
  bucket = aws_s3_bucket.lambda_artifacts.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "lambda_artifacts" {
  bucket                  = aws_s3_bucket.lambda_artifacts.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "terraform_state_bucket" { value = aws_s3_bucket.terraform_state.bucket }
output "lambda_artifacts_bucket" { value = aws_s3_bucket.lambda_artifacts.bucket }
