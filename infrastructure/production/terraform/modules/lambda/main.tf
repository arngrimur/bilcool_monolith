variable "function_name"         { type = string }
variable "s3_bucket"             { type = string }
variable "s3_key"                { type = string }
variable "role_arn"              { type = string }
variable "timeout"               {
  type = number
  default = 30
}
variable "memory_size"           {
  type = number
  default = 256
}
variable "environment_variables" {
  type = map(string)
  default = {}
}
variable "tags"                  {
  type = map(string)
  default = {}
}
variable "subnet_ids"            {
  type = list(string)
  default = []
}
variable "security_group_ids"    {
  type = list(string)
  default = []
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

  dynamic "vpc_config" {
    for_each = length(var.subnet_ids) > 0 ? [1] : []
    content {
      subnet_ids         = var.subnet_ids
      security_group_ids = var.security_group_ids
    }
  }

  depends_on = [aws_cloudwatch_log_group.fn]
}

output "function_arn"  { value = aws_lambda_function.fn.arn }
output "function_name" { value = aws_lambda_function.fn.function_name }
output "invoke_arn"    { value = aws_lambda_function.fn.invoke_arn }
