terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    neon = {
      source  = "kislerdm/neon"
      version = "~> 0.6"
    }
  }

  backend "s3" {
    bucket = "bilcool-terraform-state"
    key    = "production/terraform.tfstate"
    region = "eu-north-1"
  }
}

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "bilcool"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

provider "neon" {
  api_key = var.neon_api_key
}
