# BilCool — Production Terraform

Provisions the full AWS production stack: VPC + NAT Gateway, Neon PostgreSQL, DynamoDB, SNS/SQS, 9 Lambda functions, API Gateway, and EventBridge Scheduler.

## Prerequisites

- Terraform >= 1.9
- AWS credentials with admin access to the target account
- A Neon account and API key
- Two S3 buckets — created by the bootstrap step below

## Bootstrap (run once, before everything else)

The main Terraform config stores its state in S3 and deploys Lambda functions
from S3. Both buckets must exist before `terraform init` can run — they cannot
be created by the main config itself because Terraform needs the state bucket
before it can execute any resource creation.

Run the bootstrap config once from `../bootstrap/`:

```bash
cd ../bootstrap
terraform init   # uses local state — intentional
terraform apply
cd ../terraform
```

This creates:
- `bilcool-terraform-state` — remote backend for the main config (versioned + encrypted)
- `bilcool-lambda-artifacts` — where CI uploads Lambda ZIPs before `terraform apply`

The bootstrap state file (`../bootstrap/terraform.tfstate`) should be kept but
never committed to source control — add it to `.gitignore`.

## First-time setup

```bash
cp terraform.tfvars.example terraform.tfvars
# Fill in all REPLACE_ME values in terraform.tfvars

terraform init
terraform plan
terraform apply
```

## Custom domain (bilcool.areskiftet44.se)

The domain is configured in two steps because the TLS certificate requires DNS validation before it can be attached to API Gateway.

### Step 1 — Certificate validation

Run the first apply. It will pause after creating the ACM certificate:

```bash
terraform apply
```

While the apply is waiting, open a second terminal and get the validation record:

```bash
terraform output acm_validation_records
```

You will see output like:

```
{
  "bilcool.areskiftet44.se" = {
    "name"  = "_abc123def456.bilcool.areskiftet44.se."
    "type"  = "CNAME"
    "value" = "_xyz789.acm-validations.aws."
  }
}
```

Log in to [Loopia](https://www.loopia.se/loopiakundzon/) and add that CNAME record under `areskiftet44.se`:

| Type  | Name (relative)            | Value                          | TTL  |
|-------|----------------------------|--------------------------------|------|
| CNAME | `_abc123def456.bilcool`    | `_xyz789.acm-validations.aws.` | 3600 |

ACM typically validates within 1–2 minutes. The `terraform apply` will complete automatically once validation succeeds.

### Step 2 — Point the domain at API Gateway

After `terraform apply` completes, get the API Gateway hostname:

```bash
terraform output api_gateway_cname_target
```

You will see something like:

```
d-a1b2c3d4e5.execute-api.eu-north-1.amazonaws.com
```

Add a second CNAME in Loopia under `areskiftet44.se`:

| Type  | Name (relative) | Value                                                   | TTL  |
|-------|-----------------|---------------------------------------------------------|------|
| CNAME | `bilcool`       | `d-a1b2c3d4e5.execute-api.eu-north-1.amazonaws.com`    | 3600 |

DNS propagation usually takes a few minutes. Once it has propagated, `https://bilcool.areskiftet44.se` will route to the API.

## Subsequent deployments

Deploying new Lambda code does not require Terraform — upload updated ZIPs to S3 and run:

```bash
aws lambda update-function-code --function-name bilcool-production-bookings-http \
  --s3-bucket <bucket> --s3-key bookings-http.zip
```

Or trigger the `lambda-build` GitHub Actions job, which builds and uploads all ZIPs automatically.

Run `terraform apply` only when infrastructure changes (new resources, variable changes, scaling settings).

## Neon IP allowlist

All Lambda functions run in a private VPC and reach the internet through a single NAT Gateway Elastic IP. That IP is automatically added to the Neon project's IP allowlist by Terraform (`allowed_ips` on `module.neon`). No manual Neon configuration is needed.
