Build a microservice backend, for storing all events sent from the other services. 

# Purpose
`event-ledger` is a ledger for events so it is possible to track what has happened  in the systme a whole.
It also gives the abiity to replay old events in case the situation arises

# Backend
- The service will read from its AWS SQS queue.  
  - The service will subscribe to all notifications
- Events will be stored in a DynamoDB database. 
  - The messages is stored as JSON
  - ../message-broker defines all types and helpers needed for reading SQS messages
  - github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres.Message.MesageBody
has the inforamtion that needs to be stored
  - there will be índexes on the database for the fields that are used for searching
    - producer, event_type, emmited_at
  - the service needs to poll messages the manner as the other services does. Look at
  ../journal/internal/pkg/inbox for example
## Terraform
- The service will be deployed to a AWS ECS cluster.
- Dynamodb and SQS will be deployed to AWS using Terraform.
# Web
  - The service will expose a REST API for reading the events.
  - The API will be documented using Swagger.
  - The API will be served using Gin.
  - There will be a health check endpoint.
  - There will be a metrics endpoint.
  - There will an endpoint for reading the events with parameters for filtering. Filters are
    - producer
    - event_type
    - emitted_at
    - limit
    - offset
    - order_by
    - order_direction
    - event_id
    - emitted_at_gte
    - emitted_at_lte

# Testing
- The frontend will be tested using Ginkgo and Gomega.
- The backend will be tested using integration tests towards SQS and DynamoDB, 
using localstack and stretchr/testify suites.
  - use ../testing/aws/localcloud library for setting tests for SQS and DynamoDB:
- The service will be tested using unit tests.

# Technologies
- Golang 1.26
- AWS
