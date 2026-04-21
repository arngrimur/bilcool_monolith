variable "prefix"                              { type = string }
variable "aws_region"                          { type = string }
variable "scheduler_role_arn"                  { type = string }
variable "bookings_outbox_lambda_arn"          { type = string }
variable "authentication_outbox_lambda_arn"    { type = string }

resource "aws_scheduler_schedule" "bookings_outbox" {
  name       = "${var.prefix}-bookings-outbox"
  group_name = "default"
  tags       = { Service = "bookings", Component = "scheduler" }

  flexible_time_window {
    mode = "OFF"
  }

  # Every 15 seconds
  schedule_expression = "rate(1 minute)"

  target {
    arn      = var.bookings_outbox_lambda_arn
    role_arn = var.scheduler_role_arn

    retry_policy {
      maximum_retry_attempts = 1
    }
  }
}

resource "aws_lambda_permission" "scheduler_bookings" {
  statement_id  = "AllowSchedulerInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.bookings_outbox_lambda_arn
  principal     = "scheduler.amazonaws.com"
  source_arn    = aws_scheduler_schedule.bookings_outbox.arn
}

resource "aws_scheduler_schedule" "authentication_outbox" {
  name       = "${var.prefix}-authentication-outbox"
  group_name = "default"
  tags       = { Service = "authentication", Component = "scheduler" }

  flexible_time_window {
    mode = "OFF"
  }

  schedule_expression = "rate(1 minute)"

  target {
    arn      = var.authentication_outbox_lambda_arn
    role_arn = var.scheduler_role_arn

    retry_policy {
      maximum_retry_attempts = 1
    }
  }
}

resource "aws_lambda_permission" "scheduler_authentication" {
  statement_id  = "AllowSchedulerInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.authentication_outbox_lambda_arn
  principal     = "scheduler.amazonaws.com"
  source_arn    = aws_scheduler_schedule.authentication_outbox.arn
}
