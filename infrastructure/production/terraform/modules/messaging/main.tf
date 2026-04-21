variable "prefix" { type = string }
variable "aws_region" { type = string }

# ── SNS topics ────────────────────────────────────────────────────────────────

resource "aws_sns_topic" "bookings" {
  name = "${var.prefix}-bookings"
  tags = { Service = "bookings", Component = "messaging" }
}

resource "aws_sns_topic" "users" {
  name = "${var.prefix}-users"
  tags = { Service = "authentication", Component = "messaging" }
}

# ── SQS queues + DLQs ────────────────────────────────────────────────────────

resource "aws_sqs_queue" "bookings_dlq" {
  name                      = "${var.prefix}-bookings-dlq"
  message_retention_seconds = 1209600 # 14 days
  tags                      = { Service = "bookings" }
}

resource "aws_sqs_queue" "bookings" {
  name                       = "${var.prefix}-bookings"
  visibility_timeout_seconds = 30
  tags                       = { Service = "bookings" }
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.bookings_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sqs_queue" "event_ledger_dlq" {
  name                      = "${var.prefix}-event-ledger-dlq"
  message_retention_seconds = 1209600
  tags                      = { Service = "event-ledger", Component = "messaging" }
}

resource "aws_sqs_queue" "event_ledger" {
  name                       = "${var.prefix}-event-ledger"
  visibility_timeout_seconds = 30
  tags                       = { Service = "event-ledger", Component = "messaging" }
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.event_ledger_dlq.arn
    maxReceiveCount     = 5
  })
}

resource "aws_sqs_queue" "journal_dlq" {
  name                      = "${var.prefix}-journal-dlq"
  message_retention_seconds = 1209600
  tags                      = { Service = "journal", Component = "messaging" }
}

resource "aws_sqs_queue" "journal" {
  name                       = "${var.prefix}-journal"
  visibility_timeout_seconds = 30
  tags                       = { Service = "journal", Component = "messaging" }
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.journal_dlq.arn
    maxReceiveCount     = 5
  })
}

# ── SQS queue policies (allow SNS to send) ────────────────────────────────────

data "aws_iam_policy_document" "sqs_sns_send" {
  for_each = {
    bookings     = aws_sqs_queue.bookings.arn
    event_ledger = aws_sqs_queue.event_ledger.arn
    journal      = aws_sqs_queue.journal.arn
  }

  statement {
    effect    = "Allow"
    actions   = ["sqs:SendMessage"]
    resources = [each.value]
    principals {
      type        = "Service"
      identifiers = ["sns.amazonaws.com"]
    }
    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [aws_sns_topic.bookings.arn, aws_sns_topic.users.arn]
    }
  }
}

resource "aws_sqs_queue_policy" "bookings" {
  queue_url = aws_sqs_queue.bookings.id
  policy    = data.aws_iam_policy_document.sqs_sns_send["bookings"].json
}

resource "aws_sqs_queue_policy" "event_ledger" {
  queue_url = aws_sqs_queue.event_ledger.id
  policy    = data.aws_iam_policy_document.sqs_sns_send["event_ledger"].json
}

resource "aws_sqs_queue_policy" "journal" {
  queue_url = aws_sqs_queue.journal.id
  policy    = data.aws_iam_policy_document.sqs_sns_send["journal"].json
}

# ── SNS → SQS subscriptions ───────────────────────────────────────────────────

resource "aws_sns_topic_subscription" "bookings_to_bookings_queue" {
  topic_arn = aws_sns_topic.bookings.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.bookings.arn
}

resource "aws_sns_topic_subscription" "bookings_to_event_ledger_queue" {
  topic_arn = aws_sns_topic.bookings.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.event_ledger.arn
}

resource "aws_sns_topic_subscription" "bookings_to_journal_queue" {
  topic_arn = aws_sns_topic.bookings.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.journal.arn
}

resource "aws_sns_topic_subscription" "users_to_bookings_queue" {
  topic_arn = aws_sns_topic.users.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.bookings.arn
}

resource "aws_sns_topic_subscription" "users_to_event_ledger_queue" {
  topic_arn = aws_sns_topic.users.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.event_ledger.arn
}

resource "aws_sns_topic_subscription" "users_to_journal_queue" {
  topic_arn = aws_sns_topic.users.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.journal.arn
}

# ── outputs ───────────────────────────────────────────────────────────────────

output "bookings_topic_arn"     { value = aws_sns_topic.bookings.arn }
output "users_topic_arn"        { value = aws_sns_topic.users.arn }
output "bookings_queue_arn"     { value = aws_sqs_queue.bookings.arn }
output "event_ledger_queue_arn" { value = aws_sqs_queue.event_ledger.arn }
output "journal_queue_arn"      { value = aws_sqs_queue.journal.arn }
