env = "dev"
system = "bilcool"

# Interim: Lambda has no fixed egress IPs, so Postgres is open on 5432 and protected by TLS + SCRAM.
# Replace with specific /32s (this host, a NAT Elastic IP) once the apps move into the Frostmoln VPC.
# NOTE: the listener is created by hand in the portal and imported, and lifecycle.ignore_changes keeps
# Terraform from pushing allowed_cidrs (see db_access.tf), so this value is currently not applied.
db_allowed_cidrs = ["0.0.0.0/0"]
