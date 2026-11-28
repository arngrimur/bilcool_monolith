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

variable "db_host" {
  description = "Public address of the Frostmoln Postgres (output db_host of infrastructure/frostmoln/terraform)"
  type        = string
}

variable "db_port" {
  description = "Port of the Frostmoln Postgres"
  type        = number
  default     = 5432
}

variable "db_passwords" {
  description = "Role passwords for bookings, authentication and journal, as written by infrastructure/frostmoln/bootstrap/bootstrap.sh"
  type        = map(string)
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

variable "frontend_bucket_name" {
  description = "S3 bucket name for the React SPA assets"
  type        = string
  default     = "bilcool-frontend"
}

variable "mapbox_access_token" {
  description = "Mapbox API token for distance/routing calculations"
  type        = string
  sensitive   = true
}

variable "outbox_scheduler_enabled" {
  description = <<-EOT
    Runs the outbox Lambdas every 10 minutes via EventBridge Scheduler. false pauses the schedules (they are kept,
    state DISABLED) to save Lambda invocations; outbox rows then wait in the database and are published once it is
    set back to true. Paused 2026-11-28 to save free-tier usage.
  EOT
  type        = bool
  default     = false
}
