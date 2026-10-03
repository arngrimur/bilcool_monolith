resource "frostmoln_postgres_instance" "bilcool" {
  name           = "bilcool-db"
  version        = "16"
  flavor_id      = "db.gp1.small"
  storage_gb     = 50
  vpc_id         = frostmoln_vpc.main.id
  subnet_id      = frostmoln_subnet.bilcool.id
  ha_enabled     = false
  backup_enabled = true
}
