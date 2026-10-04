# One Postgres database, one schema + login role per service. The roles' default search_path
# points at their own schema, so the services need no schema-aware code: every connection
# (queries, dbmate, the outbox) lands in the service's own schema.
#
# The database, schemas and roles are created once by infrastructure/frostmoln/bootstrap/bootstrap.sh,
# which also generates the passwords. Pass them in as var.passwords (see terraform.tfvars.example).

variable "db_host" { type = string }
variable "db_port" {
  type    = number
  default = 5432
}
variable "db_name" {
  type    = string
  default = "bilcool"
}
variable "passwords" {
  description = "Login password per service role: bookings, authentication, journal"
  type        = map(string)
  sensitive   = true

  validation {
    condition     = alltrue([for s in ["bookings", "authentication", "journal"] : contains(keys(var.passwords), s)])
    error_message = "passwords must contain bookings, authentication and journal."
  }
}

locals {
  # No pooler in front of Frostmoln Postgres, so no pgbouncer=true (lib/pq would send it
  # to the server as an unknown setting). Runtime and migrate URLs are identical.
  urls = {
    for s in ["bookings", "authentication", "journal"] :
    s => "postgres://${s}:${var.passwords[s]}@${var.db_host}:${var.db_port}/${var.db_name}?sslmode=require"
  }
}

output "bookings_connection_string" {
  value     = local.urls["bookings"]
  sensitive = true
}
output "authentication_connection_string" {
  value     = local.urls["authentication"]
  sensitive = true
}
output "journal_connection_string" {
  value     = local.urls["journal"]
  sensitive = true
}
output "bookings_migrate_url" {
  value     = local.urls["bookings"]
  sensitive = true
}
output "authentication_migrate_url" {
  value     = local.urls["authentication"]
  sensitive = true
}
output "journal_migrate_url" {
  value     = local.urls["journal"]
  sensitive = true
}
