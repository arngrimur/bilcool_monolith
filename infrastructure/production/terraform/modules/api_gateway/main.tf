variable "prefix"                    { type = string }
variable "aws_region"                { type = string }
variable "aws_account"               { type = string }
variable "bookings_lambda_arn"       { type = string }
variable "authentication_lambda_arn" { type = string }
variable "event_ledger_lambda_arn"   { type = string }
variable "journal_lambda_arn"        { type = string }

resource "aws_apigatewayv2_api" "bilcool" {
  name          = "${var.prefix}-api"
  protocol_type = "HTTP"
  tags          = { Component = "api-gateway" }
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.bilcool.id
  name        = "$default"
  auto_deploy = true
  tags        = { Component = "api-gateway" }
}

# ── Lambda integrations ───────────────────────────────────────────────────────

locals {
  integrations = {
    bookings       = var.bookings_lambda_arn
    authentication = var.authentication_lambda_arn
    event_ledger   = var.event_ledger_lambda_arn
    journal        = var.journal_lambda_arn
  }
}

resource "aws_apigatewayv2_integration" "services" {
  for_each = local.integrations

  api_id                 = aws_apigatewayv2_api.bilcool.id
  integration_type       = "AWS_PROXY"
  integration_uri        = each.value
  payload_format_version = "2.0"
}

# ── Routes ────────────────────────────────────────────────────────────────────

resource "aws_apigatewayv2_route" "bookings" {
  api_id    = aws_apigatewayv2_api.bilcool.id
  route_key = "ANY /api/v1/bookings/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.services["bookings"].id}"
}

resource "aws_apigatewayv2_route" "authentication" {
  api_id    = aws_apigatewayv2_api.bilcool.id
  route_key = "ANY /api/v1/users/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.services["authentication"].id}"
}

resource "aws_apigatewayv2_route" "authentication_root" {
  api_id    = aws_apigatewayv2_api.bilcool.id
  route_key = "ANY /api/v1/users"
  target    = "integrations/${aws_apigatewayv2_integration.services["authentication"].id}"
}

resource "aws_apigatewayv2_route" "event_ledger" {
  api_id    = aws_apigatewayv2_api.bilcool.id
  route_key = "ANY /api/v1/events/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.services["event_ledger"].id}"
}

resource "aws_apigatewayv2_route" "journal" {
  api_id    = aws_apigatewayv2_api.bilcool.id
  route_key = "ANY /api/v1/journal/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.services["journal"].id}"
}

# ── Lambda permissions for API Gateway ───────────────────────────────────────

resource "aws_lambda_permission" "api_gateway" {
  for_each = {
    bookings       = var.bookings_lambda_arn
    authentication = var.authentication_lambda_arn
    event_ledger   = var.event_ledger_lambda_arn
    journal        = var.journal_lambda_arn
  }

  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = each.value
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.bilcool.execution_arn}/*/*"
}

output "invoke_url" { value = aws_apigatewayv2_stage.default.invoke_url }
