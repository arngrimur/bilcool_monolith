# ── TLS certificate (must live in us-east-1 for API Gateway) ─────────────────

resource "aws_acm_certificate" "bilcool" {
  provider          = aws.us_east_1
  domain_name       = var.domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

# Wait for DNS validation to complete before attaching the cert.
# The apply will block here until you add the CNAME in Loopia.
# Run: terraform output acm_validation_records  — then add those to Loopia.
resource "aws_acm_certificate_validation" "bilcool" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.bilcool.arn
  validation_record_fqdns = [for r in aws_acm_certificate.bilcool.domain_validation_options : r.resource_record_name]
}

# ── API Gateway custom domain ─────────────────────────────────────────────────

resource "aws_apigatewayv2_domain_name" "bilcool" {
  domain_name = var.domain_name
  tags        = { Component = "api-gateway" }

  domain_name_configuration {
    certificate_arn = aws_acm_certificate_validation.bilcool.certificate_arn
    endpoint_type   = "REGIONAL"
    security_policy = "TLS_1_2"
  }
}

resource "aws_apigatewayv2_api_mapping" "bilcool" {
  api_id      = module.api_gateway.api_id
  domain_name = aws_apigatewayv2_domain_name.bilcool.id
  stage       = "$default"
}

# ── Outputs ───────────────────────────────────────────────────────────────────

output "acm_validation_records" {
  description = "Add these CNAME records in Loopia to validate the TLS certificate"
  value = {
    for opt in aws_acm_certificate.bilcool.domain_validation_options : opt.domain_name => {
      name  = opt.resource_record_name
      type  = opt.resource_record_type
      value = opt.resource_record_value
    }
  }
}

output "api_gateway_cname_target" {
  description = "Point bilcool.areskiftet44.se CNAME to this hostname in Loopia"
  value       = aws_apigatewayv2_domain_name.bilcool.domain_name_configuration[0].target_domain_name
}
