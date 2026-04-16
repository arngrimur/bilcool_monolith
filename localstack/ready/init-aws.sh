#! /bin/bash

set -ex
REGION=(--region eu-north-1)
SLEEP_TIME=10
DLQ=dlq.json
FILE="file://${DLQ}"

function attributes() {
  cat > "$DLQ" <<EOF
{"RedrivePolicy":"{\"deadLetterTargetArn\":\"$1\",\"maxReceiveCount\":\"5\"}"}
EOF
}

function QueueArn() {
     awslocal --region eu-north-1 sqs get-queue-attributes --queue-url "$1" --attribute QueueArn --query 'Attributes.QueueArn' --output text
}


# create SNS topics
USERS_TOPIC=$(awslocal "${REGION[@]}" sns create-topic --name bilcool_users --query 'TopicArn' --output text)
BOOKINGS_TOPIC=$(awslocal "${REGION[@]}" sns create-topic --name bilcool_bookings --query 'TopicArn' --output text)

# CREATE SQS queues
# event-ledger
LEDGER_DLQ_NAME='bilcool-monolith-event_ledger_dlq'
LEDGER_DLQ=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name "$LEDGER_DLQ_NAME")
ARN=$(QueueArn "$LEDGER_DLQ")
attributes "$ARN"
EVENT_LEDGER_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text  create-queue --queue-name bilcool-monolith-event_ledger --attributes "$FILE")
EVENT_LEDGER_SQS_END_POINT=$(QueueArn "$EVENT_LEDGER_SQS_END_POINT")
awslocal "${REGION[@]}" sns subscribe --topic-arn "${BOOKINGS_TOPIC}" --protocol sqs --notification-endpoint "${EVENT_LEDGER_SQS_END_POINT}"

#bookings
BOOKINGS_DLQ_NAME='bilcool-monolith-bookings_dlq'
BOOKINGS_DLQ=$(awslocal "${REGION[@]}" sqs  --output text create-queue --queue-name "$BOOKINGS_DLQ_NAME" )
attributes "$(QueueArn "$BOOKINGS_DLQ")"
BOOKINGS_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name bilcool-monolith-bookings --attributes "$FILE")
BOOKINGS_SQS_END_POINT=$(QueueArn "$BOOKINGS_SQS_END_POINT")
awslocal "${REGION[@]}" sns subscribe --topic-arn "${USERS_TOPIC}" --protocol sqs --notification-endpoint "${BOOKINGS_SQS_END_POINT}"
awslocal "${REGION[@]}" sns subscribe --topic-arn "${BOOKINGS_TOPIC}" --protocol sqs --notification-endpoint "${BOOKINGS_SQS_END_POINT}"

#journal
JOURNAL_DLQ_NAME='bilcool-monolith-journal_dlq'
JOURNAL_DLQ=$(awslocal "${REGION[@]}" sqs  --output text create-queue --queue-name "$JOURNAL_DLQ_NAME" )
attributes "$(QueueArn "$JOURNAL_DLQ")"
JOURNAL_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name bilcool-monolith-journal --attributes "$FILE")
JOURNAL_SQS_END_POINT=$(QueueArn "$JOURNAL_SQS_END_POINT")
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${BOOKINGS_TOPIC}" --protocol sqs --notification-endpoint "${JOURNAL_SQS_END_POINT}"
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${USERS_TOPIC}" --protocol sqs --notification-endpoint "${JOURNAL_SQS_END_POINT}"
