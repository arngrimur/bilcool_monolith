terraform {
  required_providers {
    frostmoln = {
      source = "frostmoln/frostmoln"
    }
  }
}

provider "frostmoln" {
  # Credentials are read from provider arguments or environment variables.
  # Generate an API key in the portal under Settings -> API Keys.
}

#default taggar
resource "frostmoln_tenant_default_tags" "this" {
  tags = {
    env         = terraform.workspace
    app = vars.app
    fullname = vars.fullname
  }
}

# Vpc
resource "frostmoln_vpc" "main" {
  name = "bilcool-${terraform.workspace}-vpc"
  cidr = "10.0.0.0/16"
}

# Subnet
resource "frostmoln_subnet" "bilcool" {
  name   = "bilcool_net"
  vpc_id = frostmoln_vpc.main.id
  cidr   = "10.0.1.0/24"
  zone   = "falkenberg"
}

#Gateway
resource "frostmoln_gateway" "main" {
  vpc_id = frostmoln_vpc.main.id
  mode   = "public_ip"
  acknowledge_connectivity_loss = true
}

# Adressen som utgående trafik ser ut att komma från. På en gateway utan
# public_ip_id är detta plattformens adress — läs den för felsökning, aldrig
# för att publicera den.
output "gateway_source_address" {
  value = frostmoln_gateway.main.source_address
}


#security groups
resource "frostmoln_security_group" "web" {
  vpc_id = frostmoln_vpc.main.id
  name   = "web"
}

resource "frostmoln_security_group_rule" "https" {
  security_group_id = frostmoln_security_group.web.id
  direction         = "ingress"
  protocol          = "tcp"
  port_range_min    = 443
  port_range_max    = 443
  remote_cidr       = "0.0.0.0/0"
}

#database
resource "frostmoln_postgres_instance" "bilcool" {
  name             = "bilcool-db"
  version          = "16"
  flavor_id        = "db.gp1.medium"
  storage_gb       = 100
  vpc_id           = frostmoln_vpc.main.id
  subnet_id        = frostmoln_subnet.bilcool.id
  ha_enabled       = false
  backup_enabled   = true
}
