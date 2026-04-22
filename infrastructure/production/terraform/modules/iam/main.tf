variable "prefix"                     { type = string }
variable "aws_region"                 { type = string }
variable "aws_account"                { type = string }
variable "sns_topic_arns"             { type = list(string) }
variable "sqs_queue_arns"             { type = list(string) }
variable "dynamodb_table_arn"         { type = string }
variable "github_repo"                { type = string }
variable "lambda_artifacts_bucket"    { type = string }
variable "frontend_bucket_name"       { type = string }
variable "cloudfront_distribution_id" { type = string }

# ── Trust policy shared by all Lambda roles ───────────────────────────────────

data "aws_iam_policy_document" "lambda_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

# ── Postgres Lambda role (bookings, authentication, journal) ──────────────────

resource "aws_iam_role" "postgres_lambda" {
  name               = "${var.prefix}-postgres-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = { Component = "iam" }
}

resource "aws_iam_role_policy_attachment" "postgres_lambda_logs" {
  role       = aws_iam_role.postgres_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "postgres_lambda_vpc" {
  role       = aws_iam_role.postgres_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_iam_role_policy" "postgres_lambda_sns_sqs" {
  name = "sns-sqs"
  role = aws_iam_role.postgres_lambda.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["sns:Publish", "sns:ListTopics"]
        Resource = var.sns_topic_arns
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
        ]
        Resource = var.sqs_queue_arns
      },
    ]
  })
}

# ── DynamoDB Lambda role (event-ledger) ───────────────────────────────────────

resource "aws_iam_role" "dynamo_lambda" {
  name               = "${var.prefix}-dynamo-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = { Component = "iam" }
}

resource "aws_iam_role_policy_attachment" "dynamo_lambda_logs" {
  role       = aws_iam_role.dynamo_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "dynamo_lambda_vpc" {
  role       = aws_iam_role.dynamo_lambda.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_iam_role_policy" "dynamo_lambda_dynamo_sqs" {
  name = "dynamo-sqs"
  role = aws_iam_role.dynamo_lambda.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:Query",
          "dynamodb:Scan",
        ]
        Resource = [var.dynamodb_table_arn]
      },
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl",
        ]
        Resource = var.sqs_queue_arns
      },
    ]
  })
}

# ── EventBridge Scheduler role ────────────────────────────────────────────────

data "aws_iam_policy_document" "scheduler_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["scheduler.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "scheduler" {
  name               = "${var.prefix}-scheduler"
  assume_role_policy = data.aws_iam_policy_document.scheduler_assume.json
  tags               = { Component = "iam" }
}

resource "aws_iam_role_policy" "scheduler_invoke" {
  name = "invoke-lambdas"
  role = aws_iam_role.scheduler.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["lambda:InvokeFunction"]
      Resource = ["arn:aws:lambda:${var.aws_region}:${var.aws_account}:function:${var.prefix}-*-outbox"]
    }]
  })
}

# ── GitHub Actions OIDC deploy role ──────────────────────────────────────────

resource "aws_iam_openid_connect_provider" "github" {
  url             = "https://token.actions.githubusercontent.com"
  client_id_list  = ["sts.amazonaws.com"]
  thumbprint_list = ["6938fd4d98bab03faadb97b34396831e3780aea1"]
  tags            = { Component = "iam" }
}

data "aws_iam_policy_document" "github_deploy_assume" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values = [
        "repo:${var.github_repo}:ref:refs/heads/main",
        "repo:${var.github_repo}:ref:refs/tags/*",
      ]
    }
  }
}

resource "aws_iam_role" "github_deploy" {
  name               = "${var.prefix}-github-deploy"
  assume_role_policy = data.aws_iam_policy_document.github_deploy_assume.json
  tags               = { Component = "iam" }
}

resource "aws_iam_role_policy" "github_deploy" {
  name = "lambda-deploy"
  role = aws_iam_role.github_deploy.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:GetObject"]
        Resource = "arn:aws:s3:::${var.lambda_artifacts_bucket}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = "arn:aws:s3:::${var.lambda_artifacts_bucket}"
      },
      {
        Effect   = "Allow"
        Action   = ["lambda:UpdateFunctionCode"]
        Resource = "arn:aws:lambda:${var.aws_region}:${var.aws_account}:function:${var.prefix}-*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject", "s3:DeleteObject", "s3:GetObject"]
        Resource = "arn:aws:s3:::${var.frontend_bucket_name}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = "arn:aws:s3:::${var.frontend_bucket_name}"
      },
      {
        Effect   = "Allow"
        Action   = ["cloudfront:CreateInvalidation"]
        Resource = "arn:aws:cloudfront::${var.aws_account}:distribution/${var.cloudfront_distribution_id}"
      },
    ]
  })
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "postgres_lambda_role_arn"  { value = aws_iam_role.postgres_lambda.arn }
output "dynamo_lambda_role_arn"    { value = aws_iam_role.dynamo_lambda.arn }
output "scheduler_role_arn"        { value = aws_iam_role.scheduler.arn }
output "github_deploy_role_arn"    { value = aws_iam_role.github_deploy.arn }
