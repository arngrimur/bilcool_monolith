variable "function_name"         { type = string }
variable "s3_bucket"             { type = string }
variable "s3_key"                { type = string }
variable "role_arn"              { type = string }
variable "timeout" {
  type    = number
  default = 30
}
variable "memory_size" {
  type    = number
  default = 256
}
variable "environment_variables" {
  type    = map(string)
  default = {}
}
variable "tags" {
  type    = map(string)
  default = {}
}
variable "create_function_url" {
  type    = bool
  default = false
}

resource "aws_cloudwatch_log_group" "fn" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = 30
  tags              = var.tags
}

resource "aws_lambda_function" "fn" {
  function_name = var.function_name
  s3_bucket     = var.s3_bucket
  s3_key        = var.s3_key
  role          = var.role_arn
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = var.timeout
  memory_size   = var.memory_size
  tags          = var.tags

  environment {
    variables = var.environment_variables
  }

  depends_on = [aws_cloudwatch_log_group.fn]
}

resource "aws_lambda_function_url" "fn" {
  count              = var.create_function_url ? 1 : 0
  function_name      = aws_lambda_function.fn.function_name
  authorization_type = "NONE"

  cors {
    allow_origins = ["*"]
    allow_methods = ["*"]
    allow_headers = ["*"]
  }
}

resource "aws_lambda_permission" "function_url_public" {
  count                  = var.create_function_url ? 1 : 0
  statement_id           = "FunctionURLAllowPublicAccess"
  action                 = "lambda:InvokeFunctionUrl"
  function_name          = aws_lambda_function.fn.function_name
  principal              = "*"
  function_url_auth_type = "NONE"
}

resource "aws_lambda_permission" "invoke_public" {
  count         = var.create_function_url ? 1 : 0
  statement_id  = "InvokePublicAccess"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.fn.function_name
  principal     = "*"
}

output "function_arn"        { value = aws_lambda_function.fn.arn }
output "function_name"       { value = aws_lambda_function.fn.function_name }
output "invoke_arn"          { value = aws_lambda_function.fn.invoke_arn }
output "function_url"        { value = var.create_function_url ? aws_lambda_function_url.fn[0].function_url : "" }
output "function_url_domain" {
  value = var.create_function_url ? replace(replace(aws_lambda_function_url.fn[0].function_url, "https://", ""), "/", "") : ""
}
