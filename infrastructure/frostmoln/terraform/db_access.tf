# Public reachability for the Postgres instance.
#
# The instance itself only has a private IP in the subnet. A public L4 (TCP) load
# balancer forwards 5432 to it. L4 passes TLS straight through to Postgres and
# preserves the client source IP, so the listener's allowed_cidrs is a real allowlist.

variable "db_allowed_cidrs" {
  description = "CIDRs allowed to reach Postgres on 5432. AWS Lambda has no fixed egress IPs, so reaching it from Lambda means 0.0.0.0/0 (TLS + SCRAM only)."
  type        = list(string)

  validation {
    condition     = length(var.db_allowed_cidrs) > 0
    error_message = "At least one CIDR is required (the listener is deny-by-default)."
  }
}

# Releasing this address is irreversible and it ends up in connection strings, so protect it.
resource "frostmoln_public_ip" "db" {
  lifecycle {
    prevent_destroy = true
  }
}

resource "frostmoln_load_balancer" "db" {
  name         = "bilcool-${terraform.workspace}-db-lb"
  vpc_id       = frostmoln_vpc.main.id
  subnet_id    = frostmoln_subnet.bilcool.id
  type         = "l4"
  scheme       = "public"
  public_ip_id = frostmoln_public_ip.db.id

  # Terraform cannot see that an attached public IP needs the VPC gateway first.
  depends_on = [frostmoln_gateway.main]
}

resource "frostmoln_lb_listener" "db" {
  load_balancer_id = frostmoln_load_balancer.db.id
  name             = "postgres"
  protocol         = "tcp"
  protocol_port    = 5432
  allowed_cidrs    = var.db_allowed_cidrs
}

resource "frostmoln_lb_pool" "db" {
  load_balancer_id = frostmoln_load_balancer.db.id
  listener_id      = frostmoln_lb_listener.db.id
  name             = "postgres"
  protocol         = "tcp"
  lb_algorithm     = "source_ip_port" # the only algorithm an l4 load balancer accepts
}

resource "frostmoln_lb_member" "db" {
  load_balancer_id = frostmoln_load_balancer.db.id
  pool_id          = frostmoln_lb_pool.db.id
  name             = "bilcool-db"
  address          = frostmoln_postgres_instance.bilcool.private_ip
  protocol_port    = frostmoln_postgres_instance.bilcool.port
  subnet_id        = frostmoln_subnet.bilcool.id
}

resource "frostmoln_lb_health_monitor" "db" {
  load_balancer_id = frostmoln_load_balancer.db.id
  pool_id          = frostmoln_lb_pool.db.id
  type             = "tcp"
  delay            = 10
  timeout          = 5
  max_retries      = 3
}

output "db_host" {
  description = "Public address for DATABASE_URL (host part)."
  value       = frostmoln_public_ip.db.address
}

output "db_port" {
  value = frostmoln_lb_listener.db.protocol_port
}

output "db_admin_username" {
  description = "Admin user for the one-time bootstrap. The password is never exposed by the provider: take it from the portal."
  value       = frostmoln_postgres_instance.bilcool.admin_username
}
