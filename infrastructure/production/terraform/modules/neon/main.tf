terraform {
  required_providers {
    neon = {
      source  = "kislerdm/neon"
      version = "~> 0.6"
    }
  }
}

variable "environment"  { type = string }
variable "neon_api_key" {
  type      = string
  sensitive = true
}

resource "neon_project" "bilcool" {
  name      = "bilcool-${var.environment}"
  region_id = "aws-eu-central-1"

  history_retention_seconds = 21600
}

locals {
  host_parts  = split(".", neon_project.bilcool.database_host)
  pooler_host = join(".", concat(
    ["${local.host_parts[0]}-pooler"],
    slice(local.host_parts, 1, length(local.host_parts))
  ))
}

# ── bookings database ─────────────────────────────────────────────────────────

resource "neon_role" "bookings" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "bookings"
}

resource "neon_database" "bookings" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "bookings"
  owner_name = neon_role.bookings.name
}

# ── authentication database ───────────────────────────────────────────────────

resource "neon_role" "authentication" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "authentication"
}

resource "neon_database" "authentication" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "authentication"
  owner_name = neon_role.authentication.name
}

# ── journal database ──────────────────────────────────────────────────────────

resource "neon_role" "journal" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "journal"
}

resource "neon_database" "journal" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_project.bilcool.default_branch_id
  name       = "journal"
  owner_name = neon_role.journal.name
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "bookings_connection_string" {
  value     = "postgres://${neon_role.bookings.name}:${neon_role.bookings.password}@${local.pooler_host}/bookings?sslmode=require&pgbouncer=true"
  sensitive = true
}

output "authentication_connection_string" {
  value     = "postgres://${neon_role.authentication.name}:${neon_role.authentication.password}@${local.pooler_host}/authentication?sslmode=require&pgbouncer=true"
  sensitive = true
}

output "journal_connection_string" {
  value     = "postgres://${neon_role.journal.name}:${neon_role.journal.password}@${local.pooler_host}/journal?sslmode=require&pgbouncer=true"
  sensitive = true
}

output "bookings_migrate_url" {
  value     = "postgres://${neon_role.bookings.name}:${neon_role.bookings.password}@${neon_project.bilcool.database_host}/bookings?sslmode=require"
  sensitive = true
}

output "authentication_migrate_url" {
  value     = "postgres://${neon_role.authentication.name}:${neon_role.authentication.password}@${neon_project.bilcool.database_host}/authentication?sslmode=require"
  sensitive = true
}

output "journal_migrate_url" {
  value     = "postgres://${neon_role.journal.name}:${neon_role.journal.password}@${neon_project.bilcool.database_host}/journal?sslmode=require"
  sensitive = true
}
