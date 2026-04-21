variable "prefix" { type = string }

resource "aws_dynamodb_table" "event_ledger" {
  name         = "${var.prefix}-event-ledger"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "EventId"

  attribute {
    name = "EventId"
    type = "S"
  }

  ttl {
    attribute_name = "ExpiresAt"
    enabled        = true
  }

  tags = {
    Name      = "${var.prefix}-event-ledger"
    Service   = "event-ledger"
    Component = "dynamodb"
  }
}

output "table_name" { value = aws_dynamodb_table.event_ledger.name }
output "table_arn"  { value = aws_dynamodb_table.event_ledger.arn }
