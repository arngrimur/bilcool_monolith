output "bookings_function_url" {
  description = "Lambda Function URL for the bookings HTTP handler"
  value       = module.bookings_http.function_url
}

output "authentication_function_url" {
  description = "Lambda Function URL for the authentication HTTP handler"
  value       = module.authentication_http.function_url
}

output "event_ledger_function_url" {
  description = "Lambda Function URL for the event-ledger HTTP handler"
  value       = module.event_ledger_http.function_url
}

output "journal_function_url" {
  description = "Lambda Function URL for the journal HTTP handler"
  value       = module.journal_http.function_url
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

output "bookings_migrate_url" {
  description = "Set this as the BOOKINGS_MIGRATE_URL secret in GitHub repository settings"
  value       = module.neon.bookings_migrate_url
  sensitive   = true
}

output "authentication_migrate_url" {
  description = "Set this as the AUTHENTICATION_MIGRATE_URL secret in GitHub repository settings"
  value       = module.neon.authentication_migrate_url
  sensitive   = true
}

output "journal_migrate_url" {
  description = "Set this as the JOURNAL_MIGRATE_URL secret in GitHub repository settings"
  value       = module.neon.journal_migrate_url
  sensitive   = true
}

output "cloudfront_domain_name" {
  description = "Point bilcool.areskiftet44.se CNAME to this hostname in Loopia"
  value       = module.frontend.cloudfront_domain_name
}

output "cloudfront_distribution_id" {
  description = "Set this as the CLOUDFRONT_DISTRIBUTION_ID secret in GitHub repository settings"
  value       = module.frontend.distribution_id
}

output "frontend_bucket_name" {
  value = module.frontend.bucket_name
}

output "acm_validation_records" {
  description = "Add these CNAME records in Loopia to validate the TLS certificate"
  value       = module.frontend.acm_validation_records
}
