terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

# One Postgres database, one schema + login role per service. The roles' default search_path
# points at their own schema, so the services need no schema-aware code: every connection
# (queries, dbmate, the outbox) lands in the service's own schema.

variable "db_host" { type = string }
variable "db_port" {
  type    = number
  default = 5432
}
variable "db_name" {
  type    = string
  default = "bilcool"
}

locals {
  services = toset(["bookings", "authentication", "journal"])
}

resource "random_password" "service" {
  for_each = local.services
  length   = 40
  special  = false # keeps the password safe to embed in a URL
}

locals {
  # No pooler in front of Frostmoln Postgres, so no pgbouncer=true (lib/pq would send it
  # to the server as an unknown setting). Runtime and migrate URLs are identical.
  urls = {
    for s in local.services :
    s => "postgres://${s}:${random_password.service[s].result}@${var.db_host}:${var.db_port}/${var.db_name}?sslmode=require"
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

# Run once as the Frostmoln admin user, connected to the maintenance database. The schemas
# must exist before the migrate Lambdas run (terraform does not run this).
output "bootstrap_sql" {
  sensitive = true
  value     = <<-EOT
    SELECT 'CREATE DATABASE ${var.db_name}'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${var.db_name}')\gexec
    \connect ${var.db_name}
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;
    REVOKE CREATE ON SCHEMA public FROM PUBLIC;
    REVOKE ALL ON DATABASE ${var.db_name} FROM PUBLIC;
    %{for s in sort(tolist(local.services))~}
    DO $$ BEGIN
      IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${s}') THEN
        CREATE ROLE ${s} LOGIN PASSWORD '${random_password.service[s].result}';
      ELSE
        ALTER ROLE ${s} PASSWORD '${random_password.service[s].result}';
      END IF;
    END $$;
    CREATE SCHEMA IF NOT EXISTS ${s} AUTHORIZATION ${s};
    ALTER ROLE ${s} SET search_path = ${s}, public;
    GRANT CONNECT ON DATABASE ${var.db_name} TO ${s};
    %{endfor~}
  EOT
}
