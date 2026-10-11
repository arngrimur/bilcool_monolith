#!/bin/bash
# LocalStack bootstrap: SNS topics, SQS queues (+ DLQs) and subscriptions. Mirrors infrastructure/production/terraform
# (both topics fan out to all three queues). The DynamoDB table is created by the event_ledger service itself.
# Queue names are what message_broker/pkg/inbox/sqs/queues.go looks up.
set -ex
REGION=(--region eu-north-1)
DLQ=dlq.json
FILE="file://${DLQ}"

function attributes() {
  cat > "$DLQ" <<EOF
{"RedrivePolicy":"{\"deadLetterTargetArn\":\"$1\",\"maxReceiveCount\":\"5\"}"}
EOF
}

function QueueArn() {
  awslocal "${REGION[@]}" sqs get-queue-attributes --queue-url "$1" --attribute-names QueueArn --query 'Attributes.QueueArn' --output text
}

USERS_TOPIC=$(awslocal "${REGION[@]}" sns create-topic --name bilcool_users --query 'TopicArn' --output text)
BOOKINGS_TOPIC=$(awslocal "${REGION[@]}" sns create-topic --name bilcool_bookings --query 'TopicArn' --output text)

for q in event_ledger bookings journal; do
  DLQ_URL=$(awslocal "${REGION[@]}" sqs create-queue --queue-name "bilcool-monolith-${q}_dlq" --output text --query QueueUrl)
  attributes "$(QueueArn "$DLQ_URL")"
  Q_URL=$(awslocal "${REGION[@]}" sqs create-queue --queue-name "bilcool-monolith-${q}" --attributes "$FILE" --output text --query QueueUrl)
  for topic in "$USERS_TOPIC" "$BOOKINGS_TOPIC"; do
    awslocal "${REGION[@]}" sns subscribe --topic-arn "$topic" --protocol sqs --notification-endpoint "$(QueueArn "$Q_URL")"
  done
done
