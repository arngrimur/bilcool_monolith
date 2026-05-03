variable "prefix" { type = string }

resource "aws_dynamodb_table" "event_ledger" {
  name         = "${var.prefix}-event-ledger"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "event_id"
  range_key    = "emitted_at"

  attribute {
    name = "event_id"
    type = "S"
  }

  attribute {
    name = "emitted_at"
    type = "S"
  }

  attribute {
    name = "producer"
    type = "S"
  }

  attribute {
    name = "event_type"
    type = "S"
  }

  global_secondary_index {
    name            = "producer-emitted_at-index"
    hash_key        = "producer"
    range_key       = "emitted_at"
    projection_type = "ALL"
  }

  global_secondary_index {
    name            = "event_type-emitted_at-index"
    hash_key        = "event_type"
    range_key       = "emitted_at"
    projection_type = "ALL"
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
