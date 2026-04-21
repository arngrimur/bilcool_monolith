variable "aws_region" {
  description = "AWS region for all resources"
  type        = string
  default     = "eu-north-1"
}

variable "environment" {
  description = "Deployment environment name"
  type        = string
  default     = "production"
}

variable "neon_api_key" {
  description = "Neon API key for database provisioning"
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "Secret used to sign JWT tokens"
  type        = string
  sensitive   = true
}

variable "brevo_api_key" {
  description = "Brevo transactional email API key"
  type        = string
  sensitive   = true
}

variable "from_email" {
  description = "Sender address for transactional emails"
  type        = string
}

variable "webauthn_rp_id" {
  description = "WebAuthn relying party ID (typically the domain)"
  type        = string
}

variable "webauthn_rp_origins" {
  description = "Comma-separated list of allowed WebAuthn origins"
  type        = string
}

variable "webauthn_display_name" {
  description = "WebAuthn relying party display name"
  type        = string
  default     = "BilCool"
}

variable "lambda_artifacts_bucket" {
  description = "S3 bucket holding Lambda ZIP artifacts"
  type        = string
}

variable "domain_name" {
  description = "Public hostname for the API (e.g. bilcool.areskiftet44.se)"
  type        = string
  default     = "bilcool.areskiftet44.se"
}
