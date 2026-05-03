terraform {
  required_providers {
    aws = {
      source                = "hashicorp/aws"
      configuration_aliases = [aws.us_east_1]
    }
  }
}

variable "prefix"                             { type = string }
variable "domain_name"                        { type = string }
variable "bucket_name"                        { type = string }
variable "bookings_function_url_domain"       { type = string }
variable "authentication_function_url_domain" { type = string }
variable "event_ledger_function_url_domain"   { type = string }
variable "journal_function_url_domain"        { type = string }

# ── S3 bucket ─────────────────────────────────────────────────────────────────

resource "aws_s3_bucket" "frontend" {
  bucket        = var.bucket_name
  force_destroy = true
  tags          = { Component = "frontend" }
}

resource "aws_s3_bucket_versioning" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_public_access_block" "frontend" {
  bucket                  = aws_s3_bucket.frontend.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# ── CloudFront Origin Access Control ─────────────────────────────────────────

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "${var.prefix}-frontend"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_s3_bucket_policy" "frontend" {
  bucket = aws_s3_bucket.frontend.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.frontend.arn}/*"
      Condition = {
        StringEquals = {
          "AWS:SourceArn" = aws_cloudfront_distribution.frontend.arn
        }
      }
    }]
  })
}

# ── ACM certificate (must be in us-east-1 for CloudFront) ────────────────────

resource "aws_acm_certificate" "frontend" {
  provider          = aws.us_east_1
  domain_name       = var.domain_name
  validation_method = "DNS"
  tags              = { Component = "frontend" }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_acm_certificate_validation" "frontend" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.frontend.arn
  validation_record_fqdns = [for r in aws_acm_certificate.frontend.domain_validation_options : r.resource_record_name]
}

# ── CloudFront Function: SPA path rewriting ───────────────────────────────────

resource "aws_cloudfront_function" "spa_router" {
  name    = "${var.prefix}-spa-router"
  runtime = "cloudfront-js-2.0"
  publish = true
  code    = <<-EOF
    function handler(event) {
      var uri = event.request.uri;
      if (!/\.[^/]+$/.test(uri)) {
        event.request.uri = '/index.html';
      }
      return event.request;
    }
  EOF
}

# ── CloudFront distribution ───────────────────────────────────────────────────

locals {
  s3_origin_id              = "s3-frontend"
  bookings_origin_id        = "bookings-lambda"
  authentication_origin_id  = "authentication-lambda"
  event_ledger_origin_id    = "event-ledger-lambda"
  journal_origin_id         = "journal-lambda"

  lambda_custom_origin = {
    http_port              = 80
    https_port             = 443
    origin_protocol_policy = "https-only"
    origin_ssl_protocols   = ["TLSv1.2"]
  }

  api_forwarded_headers = ["Authorization", "Content-Type", "Accept", "Origin", "Correlation-Id", "Access-Control-Request-Headers", "Access-Control-Request-Method"]
}

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  aliases             = [var.domain_name]
  tags                = { Component = "frontend" }

  # ── Origins ────────────────────────────────────────────────────────────────

  origin {
    origin_id                = local.s3_origin_id
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  origin {
    origin_id   = local.bookings_origin_id
    domain_name = var.bookings_function_url_domain
    custom_origin_config {
      http_port              = local.lambda_custom_origin.http_port
      https_port             = local.lambda_custom_origin.https_port
      origin_protocol_policy = local.lambda_custom_origin.origin_protocol_policy
      origin_ssl_protocols   = local.lambda_custom_origin.origin_ssl_protocols
    }
  }

  origin {
    origin_id   = local.authentication_origin_id
    domain_name = var.authentication_function_url_domain
    custom_origin_config {
      http_port              = local.lambda_custom_origin.http_port
      https_port             = local.lambda_custom_origin.https_port
      origin_protocol_policy = local.lambda_custom_origin.origin_protocol_policy
      origin_ssl_protocols   = local.lambda_custom_origin.origin_ssl_protocols
    }
  }

  origin {
    origin_id   = local.event_ledger_origin_id
    domain_name = var.event_ledger_function_url_domain
    custom_origin_config {
      http_port              = local.lambda_custom_origin.http_port
      https_port             = local.lambda_custom_origin.https_port
      origin_protocol_policy = local.lambda_custom_origin.origin_protocol_policy
      origin_ssl_protocols   = local.lambda_custom_origin.origin_ssl_protocols
    }
  }

  origin {
    origin_id   = local.journal_origin_id
    domain_name = var.journal_function_url_domain
    custom_origin_config {
      http_port              = local.lambda_custom_origin.http_port
      https_port             = local.lambda_custom_origin.https_port
      origin_protocol_policy = local.lambda_custom_origin.origin_protocol_policy
      origin_ssl_protocols   = local.lambda_custom_origin.origin_ssl_protocols
    }
  }

  # ── Default: serve SPA from S3 ─────────────────────────────────────────────

  default_cache_behavior {
    target_origin_id       = local.s3_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = false
      cookies { forward = "none" }
    }

    min_ttl     = 0
    default_ttl = 86400
    max_ttl     = 31536000

    function_association {
      event_type   = "viewer-request"
      function_arn = aws_cloudfront_function.spa_router.arn
    }
  }

  # ── /api/v1/bookings* → bookings Lambda ────────────────────────────────────

  ordered_cache_behavior {
    path_pattern           = "/api/v1/bookings*"
    target_origin_id       = local.bookings_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = true
      headers      = local.api_forwarded_headers
      cookies { forward = "all" }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  # ── /api/v1/users* → authentication Lambda ─────────────────────────────────

  ordered_cache_behavior {
    path_pattern           = "/api/v1/users*"
    target_origin_id       = local.authentication_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = true
      headers      = local.api_forwarded_headers
      cookies { forward = "all" }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  # ── /api/v1/events* → event-ledger Lambda ──────────────────────────────────

  ordered_cache_behavior {
    path_pattern           = "/api/v1/events*"
    target_origin_id       = local.event_ledger_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = true
      headers      = local.api_forwarded_headers
      cookies { forward = "all" }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  # ── /api/v1/journal* → journal Lambda ──────────────────────────────────────

  ordered_cache_behavior {
    path_pattern           = "/api/v1/journal*"
    target_origin_id       = local.journal_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true

    forwarded_values {
      query_string = true
      headers      = local.api_forwarded_headers
      cookies { forward = "all" }
    }

    min_ttl     = 0
    default_ttl = 0
    max_ttl     = 0
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.frontend.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }
}

# ── Outputs ───────────────────────────────────────────────────────────────────

output "cloudfront_domain_name" { value = aws_cloudfront_distribution.frontend.domain_name }
output "distribution_id"        { value = aws_cloudfront_distribution.frontend.id }
output "bucket_name"            { value = aws_s3_bucket.frontend.bucket }

output "acm_validation_records" {
  description = "Add these CNAME records in Loopia to validate the TLS certificate"
  value = {
    for opt in aws_acm_certificate.frontend.domain_validation_options : opt.domain_name => {
      name  = opt.resource_record_name
      type  = opt.resource_record_type
      value = opt.resource_record_value
    }
  }
}
