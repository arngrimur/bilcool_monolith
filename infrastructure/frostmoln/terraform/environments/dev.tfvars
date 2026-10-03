env = "dev"
system = "bilcool"

# Interim: Lambda has no fixed egress IPs, so Postgres is open on 5432 and protected by TLS + SCRAM.
# Replace with specific /32s (this host, a NAT Elastic IP) once the apps move into the Frostmoln VPC.
# Written as two halves because the platform does not round-trip a literal 0.0.0.0/0 on the listener
# (it reads back as []), which makes the provider fail with "inconsistent result after apply".
db_allowed_cidrs = ["81.225.208.217/32"]
