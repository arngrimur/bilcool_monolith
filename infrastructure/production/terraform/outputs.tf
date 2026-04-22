output "api_gateway_url" {
  description = "Base URL of the HTTP API Gateway"
  value       = module.api_gateway.invoke_url
}

output "bookings_http_lambda_arn" {
  value = module.bookings_http.function_arn
}

output "authentication_http_lambda_arn" {
  value = module.authentication_http.function_arn
}

output "event_ledger_http_lambda_arn" {
  value = module.event_ledger_http.function_arn
}

output "journal_http_lambda_arn" {
  value = module.journal_http.function_arn
}

output "github_deploy_role_arn" {
  description = "Set this as the AWS_DEPLOY_ROLE_ARN secret in GitHub repository settings"
  value       = module.iam.github_deploy_role_arn
}
