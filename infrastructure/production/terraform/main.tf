locals {
  prefix = "bilcool-${var.environment}"
}

# ── Neon databases ────────────────────────────────────────────────────────────

module "neon" {
  source      = "./modules/neon"
  environment = var.environment
  neon_api_key = var.neon_api_key
}

# ── Messaging (SNS + SQS) ─────────────────────────────────────────────────────

module "messaging" {
  source      = "./modules/messaging"
  prefix      = local.prefix
  aws_region  = var.aws_region
}

# ── DynamoDB ──────────────────────────────────────────────────────────────────

module "dynamodb" {
  source      = "./modules/dynamodb"
  prefix      = local.prefix
}

# ── IAM roles ─────────────────────────────────────────────────────────────────

module "iam" {
  source      = "./modules/iam"
  prefix      = local.prefix
  aws_region  = var.aws_region
  aws_account = data.aws_caller_identity.current.account_id

  sns_topic_arns = [
    module.messaging.bookings_topic_arn,
    module.messaging.users_topic_arn,
  ]
  sqs_queue_arns = [
    module.messaging.bookings_queue_arn,
    module.messaging.event_ledger_queue_arn,
    module.messaging.journal_queue_arn,
  ]
  dynamodb_table_arn = module.dynamodb.table_arn
}

data "aws_caller_identity" "current" {}

# ── Lambda: bookings ──────────────────────────────────────────────────────────

module "bookings_http" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-bookings-http"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "bookings-http.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  environment_variables = {
    DATABASE_URL = module.neon.bookings_connection_string
    OUTBOX_MODE  = "polling"
  }
}

module "bookings_sqs" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-bookings-sqs"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "bookings-sqs.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  environment_variables = {
    DATABASE_URL = module.neon.bookings_connection_string
  }
}

module "bookings_outbox" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-bookings-outbox"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "bookings-outbox.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  timeout       = 60
  environment_variables = {
    DATABASE_URL = module.neon.bookings_connection_string
  }
}

resource "aws_lambda_event_source_mapping" "bookings_sqs" {
  event_source_arn                   = module.messaging.bookings_queue_arn
  function_name                      = module.bookings_sqs.function_arn
  batch_size                         = 10
  function_response_types            = ["ReportBatchItemFailures"]
}

# ── Lambda: authentication ────────────────────────────────────────────────────

module "authentication_http" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-authentication-http"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "authentication-http.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  environment_variables = {
    DATABASE_URL          = module.neon.authentication_connection_string
    OUTBOX_MODE           = "polling"
    JWT_SECRET            = var.jwt_secret
    FROM_EMAIL            = var.from_email
    BREVO_API_KEY         = var.brevo_api_key
    WEBAUTHN_RP_ID        = var.webauthn_rp_id
    WEBAUTHN_RP_ORIGINS   = var.webauthn_rp_origins
    WEBAUTHN_DISPLAY_NAME = var.webauthn_display_name
  }
}

module "authentication_outbox" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-authentication-outbox"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "authentication-outbox.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  timeout       = 60
  environment_variables = {
    DATABASE_URL  = module.neon.authentication_connection_string
    JWT_SECRET    = var.jwt_secret
    FROM_EMAIL    = var.from_email
    BREVO_API_KEY = var.brevo_api_key
    WEBAUTHN_RP_ID      = var.webauthn_rp_id
    WEBAUTHN_RP_ORIGINS = var.webauthn_rp_origins
  }
}

# ── Lambda: event-ledger ──────────────────────────────────────────────────────

module "event_ledger_http" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-event-ledger-http"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "event-ledger-http.zip"
  role_arn      = module.iam.dynamo_lambda_role_arn
  environment_variables = {
    DYNAMO_TABLE_NAME = module.dynamodb.table_name
  }
}

module "event_ledger_sqs" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-event-ledger-sqs"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "event-ledger-sqs.zip"
  role_arn      = module.iam.dynamo_lambda_role_arn
  environment_variables = {
    DYNAMO_TABLE_NAME = module.dynamodb.table_name
  }
}

resource "aws_lambda_event_source_mapping" "event_ledger_sqs" {
  event_source_arn        = module.messaging.event_ledger_queue_arn
  function_name           = module.event_ledger_sqs.function_arn
  batch_size              = 10
  function_response_types = ["ReportBatchItemFailures"]
}

# ── Lambda: journal ───────────────────────────────────────────────────────────

module "journal_http" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-journal-http"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "journal-http.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  environment_variables = {
    DATABASE_URL = module.neon.journal_connection_string
  }
}

module "journal_sqs" {
  source        = "./modules/lambda"
  function_name = "${local.prefix}-journal-sqs"
  s3_bucket     = var.lambda_artifacts_bucket
  s3_key        = "journal-sqs.zip"
  role_arn      = module.iam.postgres_lambda_role_arn
  environment_variables = {
    DATABASE_URL = module.neon.journal_connection_string
  }
}

resource "aws_lambda_event_source_mapping" "journal_sqs" {
  event_source_arn        = module.messaging.journal_queue_arn
  function_name           = module.journal_sqs.function_arn
  batch_size              = 10
  function_response_types = ["ReportBatchItemFailures"]
}

# ── API Gateway ───────────────────────────────────────────────────────────────

module "api_gateway" {
  source      = "./modules/api_gateway"
  prefix      = local.prefix
  aws_region  = var.aws_region
  aws_account = data.aws_caller_identity.current.account_id

  bookings_lambda_arn       = module.bookings_http.function_arn
  authentication_lambda_arn = module.authentication_http.function_arn
  event_ledger_lambda_arn   = module.event_ledger_http.function_arn
  journal_lambda_arn        = module.journal_http.function_arn
}

# ── EventBridge Scheduler: outbox dispatchers ─────────────────────────────────

module "outbox_scheduler" {
  source      = "./modules/scheduler"
  prefix      = local.prefix
  aws_region  = var.aws_region

  bookings_outbox_lambda_arn       = module.bookings_outbox.function_arn
  authentication_outbox_lambda_arn = module.authentication_outbox.function_arn
  scheduler_role_arn               = module.iam.scheduler_role_arn
}
