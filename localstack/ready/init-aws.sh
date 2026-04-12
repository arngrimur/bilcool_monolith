#! /bin/bash

function attributes() {
  printf '{"RedrivePolicy": \"{\\"deadLetterTargetArn\\":\\"%s\\"}"}' "$1" > dlq.json
  echo "file://dlq.json"
}

function QueueArn() {
     awslocal --region eu-north-1 --output text sqs get-queue-attributes --queue-url "$1" --attribute QueueArn | sed 's/ATTRIBUTES\t//g'
}
REGION=(--region eu-north-1)

# create SNS topics
USERS_TOPIC=$(awslocal "${REGION[@]}" sns  create-topic --output text --name bilcool_users)
BOOKINGS_TOPIC=$(awslocal "${REGION[@]}" sns  create-topic --output text  --name bilcool_bookings)

# CREATE SQS queues
# event-ledger
LEDGER_DLQ_NAME='bilcool-monolith-event_ledger_dlq'
LEDGER_DLQ=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name "$LEDGER_DLQ_NAME")
FILE=$(attributes "$LEDGER_DLQ")
EVENT_LEDGER_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text  create-queue --queue-name bilcool-monolith-event_ledger --attributes "$FILE")
EVENT_LEDGER_SQS_END_POINT=$(QueueArn "$EVENT_LEDGER_SQS_END_POINT")
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${USERS_TOPIC}" --protocol sqs --notification-endpoint "${EVENT_LEDGER_SQS_END_POINT}"
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${BOOKINGS_TOPIC}" --protocol sqs --notification-endpoint "${EVENT_LEDGER_SQS_END_POINT}"

#bookings
BOOKINGS_DLQ_NAME='bilcool-monolith-bookings_dlq'
BOOKINGS_DLQ=$(awslocal "${REGION[@]}" sqs  --output text create-queue --queue-name "$BOOKINGS_DLQ_NAME" )
FILE=$(attributes "$BOOKINGS_DLQ")
BOOKINGS_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name bilcool-monolith-bookings --attributes "$FILE")
BOOKINGS_SQS_END_POINT=$(QueueArn "$BOOKINGS_SQS_END_POINT")
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${USERS_TOPIC}" --protocol sqs --notification-endpoint "${BOOKINGS_SQS_END_POINT}"

#journal
JOURNAL_DLQ_NAME='bilcool-monolith-journal_dlq'
JOURNAL_DLQ=$(awslocal "${REGION[@]}" sqs  --output text create-queue --queue-name "$JOURNAL_DLQ_NAME" )
FILE=$(attributes "$JOURNAL_DLQ")
JOURNAL_SQS_END_POINT=$(awslocal "${REGION[@]}" sqs --output text create-queue --queue-name bilcool-monolith-journal --attributes "$FILE")
JOURNAL_SQS_END_POINT=$(QueueArn "$JOURNAL_SQS_END_POINT")
awslocal "${REGION[@]}" sns  subscribe --topic-arn "${BOOKINGS_TOPIC}" --protocol sqs --notification-endpoint "${JOURNAL_SQS_END_POINT}"
