variable "environment" { type = string }
variable "neon_api_key" { type = string; sensitive = true }

resource "neon_project" "bilcool" {
  name      = "bilcool-${var.environment}"
  region_id = "aws-eu-north-1"
}

# ── bookings database ─────────────────────────────────────────────────────────

resource "neon_branch" "bookings" {
  project_id = neon_project.bilcool.id
  name       = "bookings"
}

resource "neon_database" "bookings" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.bookings.id
  name       = "bookings"
  owner_name = neon_role.bookings.name
}

resource "neon_role" "bookings" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.bookings.id
  name       = "bookings"
}

# ── authentication database ───────────────────────────────────────────────────

resource "neon_branch" "authentication" {
  project_id = neon_project.bilcool.id
  name       = "authentication"
}

resource "neon_database" "authentication" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.authentication.id
  name       = "authentication"
  owner_name = neon_role.authentication.name
}

resource "neon_role" "authentication" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.authentication.id
  name       = "authentication"
}

# ── journal database ──────────────────────────────────────────────────────────

resource "neon_branch" "journal" {
  project_id = neon_project.bilcool.id
  name       = "journal"
}

resource "neon_database" "journal" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.journal.id
  name       = "journal"
  owner_name = neon_role.journal.name
}

resource "neon_role" "journal" {
  project_id = neon_project.bilcool.id
  branch_id  = neon_branch.journal.id
  name       = "journal"
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "bookings_connection_string" {
  value     = "postgres://${neon_role.bookings.name}:${neon_role.bookings.password}@${neon_project.bilcool.database_host}-pooler.${neon_project.bilcool.database_host}/bookings?sslmode=require&pool_mode=transaction"
  sensitive = true
}

output "authentication_connection_string" {
  value     = "postgres://${neon_role.authentication.name}:${neon_role.authentication.password}@${neon_project.bilcool.database_host}-pooler.${neon_project.bilcool.database_host}/authentication?sslmode=require&pool_mode=transaction"
  sensitive = true
}

output "journal_connection_string" {
  value     = "postgres://${neon_role.journal.name}:${neon_role.journal.password}@${neon_project.bilcool.database_host}-pooler.${neon_project.bilcool.database_host}/journal?sslmode=require&pool_mode=transaction"
  sensitive = true
}
