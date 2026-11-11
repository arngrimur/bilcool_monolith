# Plan: three environments, all as infrastructure as code

> **Status: proposal for review. Nothing is implemented.** Written 2026-11-11.
> Section 2 lists the decisions I need from you; each has a recommendation that the rest of the plan assumes.

## 1. Target

| | **local** | **aws** (today's production) | **frostmoln** (everything on Frostmoln) |
|---|---|---|---|
| Runtime | minikube, Helm chart | AWS Lambda + Function URLs | Frostmoln managed Kubernetes, Helm chart |
| Database | Postgres 18 in Docker, beside minikube | Frostmoln Postgres via the public L4 load balancer | Frostmoln Postgres over the private VPC network |
| Messaging | LocalStack SNS/SQS in Docker, beside minikube | AWS SNS/SQS | In-cluster broker (no AWS) |
| Event ledger store | LocalStack DynamoDB | AWS DynamoDB | Postgres schema `event_ledger` |
| UI | nginx pod | S3 + CloudFront | nginx pod behind ingress (Traefik + cert-manager) |
| Mail | `MAIL_SENDER=log` | Brevo | Brevo |
| Defined by | Terraform (`stacks/local`) | Terraform (`stacks/frostmoln-base` + `stacks/aws-app`) | Terraform (`stacks/frostmoln-base` + `stacks/frostmoln-app`) |

### What is not infrastructure as code today

| Gap | Where | Fixed in |
|---|---|---|
| Database, schemas and roles are created by shell scripts, in two copies | `frostmoln/bootstrap/bootstrap.sh`, `helm/bilcool/files/db-init.sh` | Phase 2 (local), Phase 3 (Frostmoln) |
| Topics, queues and subscriptions are defined twice, once as a shell script | `helm/bilcool/files/init-aws.sh` vs `production/terraform/modules/messaging` | Phase 2 |
| `db_host` and the role passwords are copied by hand from one stack to the other | `db-credentials.env` -> `TF_VAR_db_passwords` | Phase 3 |
| The load balancer listener was created in the portal and imported | `frostmoln/terraform/db_access.tf` | Phase 3 (if the platform bug is fixed), gone in Phase 6 |
| No Kubernetes cluster, ingress or prod Helm values exist | `plan.md` "Plan 2" | Phase 5 |
| CI only runs `terraform validate`, never a plan against state | `build.yml` `terraform-plan` | Phase 3 |
| DNS is edited by hand at Loopia | - | Stays manual (see 2.6) |

## 2. Decisions for you

1. **Are `aws` and `frostmoln` two targets for the one production, or two environments running side by side?**
   Recommended: two targets for one production. They share the one Frostmoln database, and you cut over from `aws` to
   `frostmoln` when ready (Phase 6). Side by side would need a second database, a second domain and double the cost.
2. **Local IaC tool: Terraform or docker compose?**
   Recommended: Terraform (`kreuzwerker/docker`, `cyrilgdn/postgresql`, `hashicorp/aws` pointed at LocalStack,
   `hashicorp/helm`). Local then runs the *same* database and messaging modules as production, and the two shell
   scripts disappear. `minikube start` itself stays a Taskfile step.
3. **Should LocalStack also move out of minikube into Docker?** You only mentioned the database.
   Recommended: yes. Terraform on the host must reach it to create topics and queues before the Helm release is installed.
4. **Message broker for `frostmoln`.** There is no SNS/SQS there, so `message_broker` needs a second transport (Go change).
   Recommended: NATS JetStream in the cluster. A stream per topic and a durable consumer per service map directly onto
   today's topic -> queue fan-out. Alternatives: RabbitMQ (heavier to run), or a Frostmoln managed queue if one exists
   (I could not check, see section 6).
5. **Event ledger store for `frostmoln`.** No DynamoDB there.
   Recommended: a Postgres implementation in a fourth schema `event_ledger` (the `EventStore` and `EventQuerier`
   interfaces already exist).
6. **DNS.** Loopia has no maintained Terraform provider.
   Recommended: keep it manual and documented as the one exception. Alternative: delegate `bilcool.areskiftet44.se`
   to a DNS service that Terraform can manage.
7. **Role passwords when the bootstrap moves into Terraform.** They will be stored in Terraform state (S3).
   Recommended: import the existing roles and keep the current passwords (no rotation, no Lambda change), and confirm
   the state bucket is encrypted and not public.
8. **Terraform state for `frostmoln`.** Recommended: stay in the existing S3 bucket for now. "No AWS at all" would also
   need a new state backend, which depends on what Frostmoln offers.

## 3. Layout

```
infrastructure/
├── modules/                    # reusable Terraform modules
│   ├── db-bootstrap/           # schema + login role per service, search_path, grants (replaces bootstrap.sh and db-init.sh)
│   ├── messaging-sns-sqs/      # topics, queues, DLQs, subscriptions; names as variables (from production/terraform/modules/messaging)
│   ├── dynamodb/               # from production/terraform/modules
│   ├── aws-lambda/, aws-iam/, aws-frontend/, aws-scheduler/   # from production/terraform/modules
│   ├── frostmoln-network/      # VPC, subnet, gateway
│   ├── frostmoln-postgres/     # instance, plus the public load balancer behind a switch
│   ├── frostmoln-k8s/          # cluster and node pool
│   └── bilcool-release/        # helm_release of the chart
├── stacks/                     # one Terraform root per thing you apply
│   ├── local/                  # Docker Postgres + LocalStack, db-bootstrap, messaging, Helm release into minikube
│   ├── frostmoln-base/         # network, Postgres, db-bootstrap   (today: frostmoln/terraform)
│   ├── aws-app/                # Lambdas, SNS/SQS, DynamoDB, CloudFront   (today: production/terraform)
│   └── frostmoln-app/          # cluster, Traefik, cert-manager, broker, Helm release
└── helm/bilcool/               # application workloads only: services, UI, migrations, seed, ingress
```

- `aws` = `frostmoln-base` + `aws-app`. `frostmoln` = `frostmoln-base` + `frostmoln-app`.
- The app stacks read `db_host` and the role passwords from `frostmoln-base` with `terraform_remote_state`.
- The Helm chart loses its Postgres StatefulSet, `db-init` Job and LocalStack Deployment. It takes database URLs and
  messaging settings as values only, so the same chart serves `local` and `frostmoln`.

## 4. Phases (one pull request each)

### Phase 1: restructure, no behaviour change
- Move `frostmoln/terraform` -> `stacks/frostmoln-base`, `production/terraform` -> `stacks/aws-app`, modules -> `modules/`.
- **Keep the S3 backend keys unchanged** and add `moved` blocks for every changed module address.
- Update the paths in `build.yml`, `Taskfile.yml`, the READMEs and the `CLAUDE.md` files.
- Remove dead material: `production/kubernetes` (legacy), the `dump-to-local.sh` section of the Frostmoln README (the script
  no longer exists), the unused `env` variable.
- **Acceptance:** `terraform plan` on both live stacks reports no changes. You run this (it needs your credentials).

### Phase 2: local environment
- `stacks/local`: Postgres 18 container (`wal_level=logical`, named volume, port 54320) and LocalStack container (port 4566).
- `modules/db-bootstrap` creates database objects, roles and schemas; `modules/messaging-sns-sqs` creates topics and queues
  in LocalStack. Delete `db-init.sh` and `init-aws.sh`.
- Pods reach both through `host.minikube.internal`; the stack passes the URLs to the Helm release.
- Chart: remove `postgres.yaml`, `localstack.yaml` and their secrets; rename `values-dev.yaml` to `values-local.yaml`.
- `task up` becomes: start minikube, build and load images, `terraform -chdir=infrastructure/stacks/local apply`.
  `task down` destroys the release and keeps the database volume; `task down:purge` removes the volume.
  The port-forward task only needs the UI.
- Update `dump-to-minikube.sh` and `task bookings_db` (they use the pod `postgres-0` today).
- Local state is a gitignored local file. The LocalStack token and Mapbox token come from `TF_VAR_*`; `values-secrets.yaml` goes away.
- **Acceptance:** from a clean machine, `task up` gives a working login and a booking whose event reaches journal and
  event_ledger, in both `polling` and `replication` outbox modes.
- **To verify early:** that pods can reach a host-published port on Linux with the minikube Docker driver (the container
  must not bind to 127.0.0.1 only).

### Phase 3: close the IaC gaps in `aws`
- `frostmoln-base` uses `modules/db-bootstrap`; import the existing roles and schemas; delete `bootstrap.sh`
  (`check-tables.sh` stays as a verification tool). The `pgadmin` password is supplied as `TF_VAR`, never stored in the repo.
- `aws-app` reads `db_host` and passwords from remote state; `db-credentials.env` and the hand-copied variables go away.
- Retry managing the listener and `allowed_cidrs` in Terraform; if the platform still refuses, keep the documented exception.
- Queue and topic names become variables. No rename in production.
- CI: a real `terraform plan` on pull requests (read-only AWS role and a Frostmoln API key as secrets), posted as a job summary.
- **Acceptance:** plan shows no changes to Lambdas, queues or the database; `check-tables.sh` passes.

### Phase 4: make the code portable (Go, no infrastructure change)
- `message_broker`: a transport interface with the existing SNS/SQS implementation and a new NATS one, selected by
  `MESSAGING_TRANSPORT` (default `sns-sqs`). Queue and topic names from the environment, with today's names as defaults.
- `event_ledger`: a Postgres store beside DynamoDB, selected by configuration; a migrate image like the other services.
- `stacks/local` gets `-var messaging=nats` so the new path is tested in minikube before any cluster exists.
- **Acceptance:** unit and integration tests pass for both transports; the Lambda path is byte-for-byte the same configuration.

### Phase 5: `frostmoln-app`
- `modules/frostmoln-k8s`: cluster (Kubernetes 1.35) and node pool. Open from `plan.md`: control-plane tier, node flavor and count.
- Helm releases via Terraform: Traefik, cert-manager, NATS JetStream, then `bilcool` with `values-frostmoln.yaml`.
- Database over the private VPC address; `event_ledger` schema and role added in `frostmoln-base`.
- UI image switches to `nginx-unprivileged` (restricted pod security).
- CI: `helm lint` and `helm template` on pull requests; `helm-deploy-frostmoln` is replaced by a Terraform apply and stays
  behind the `DEPLOY_FROSTMOLN_K8S` variable.
- **Acceptance:** the full flow works on a temporary hostname while production keeps running on Lambda.

### Phase 6: cutover (a separate go/no-go)
- Stop the outbox schedulers and drain the SQS queues, point the Loopia record at the Frostmoln ingress address, keep
  `WEBAUTHN_RP_ID` and origins unchanged so passkeys keep working.
- After a soak period: `terraform destroy` on `aws-app`, switch off the public database load balancer, and stop CI's
  `lambda-deploy` and `ui-deploy`.
- Rollback until the destroy: point DNS back at CloudFront and re-enable the schedulers.

## 5. Risks

- **Phase 1 touches the code of the live stacks.** The `moved` blocks and the no-change plan are the safeguard; nothing is
  applied by CI.
- **Merging any phase to `main` redeploys the Lambdas and UI** (same code, so harmless, but it happens).
- **Role passwords in Terraform state** (decision 2.7).
- **In-order delivery and redelivery differ between SQS and NATS.** Consumers are already idempotent (`inbox`), so the
  integration tests in Phase 4 focus on redelivery and dead-lettering.
- **Frostmoln cannot set `wal_level`**, so `frostmoln` runs the outbox in `polling` mode, with a short interval in-cluster
  (seconds instead of today's 10 minutes).
- **The Frostmoln workspace is called `dev` but holds production.** I propose leaving the name alone (renaming recreates
  resources) and documenting it.

## 6. What I could not check from this session

- The Frostmoln provider schema: the registry is blocked here. Before Phase 5 someone must confirm the Kubernetes
  resources, whether a managed queue or S3-compatible storage exists, and whether the listener bug is fixed.
- No `terraform`, `helm` or `minikube` in this session, so plans and local runs must happen on your machine or after the
  session environment is extended.
- I did not read `environments/*.tfvars` (the repository rules mark `*.tfvars` as secret). Note that `dev.tfvars` and
  `prod.tfvars` under `frostmoln/terraform/environments` are tracked by git; worth a look to confirm they hold nothing sensitive.
