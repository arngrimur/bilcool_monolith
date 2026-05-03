package main

import (
	"context"
	"encoding/json"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/inbox"
	dynstore "github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/persistance/dynamodb"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("event-ledger-sqs"))
	log.Ctx(ctx).Info().Msg("starting event-ledger sqs lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS config")
	}

	dynamoClient := dynstore.NewClientFromConfig(awsCfg)
	store := dynstore.NewEventRepository(dynamoClient, config.DynamoTableName())

	lambda.Start(func(lambdaCtx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
		handlerCtx := log.Logger.WithContext(lambdaCtx)
		response := events.SQSEventResponse{}
		for _, record := range event.Records {
			var body brokerpostgres.MessageBody
			if err := json.Unmarshal([]byte(record.Body), &body); err != nil {
				log.Ctx(handlerCtx).Err(err).Str("message_id", record.MessageId).Msg("failed to parse message")
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
				continue
			}
			msg := brokerpostgres.Message{ReceiptHandle: record.ReceiptHandle, MessageBody: body}
			item := inbox.MessageToEventItem(msg)
			if err := store.SaveEvent(handlerCtx, item); err != nil {
				log.Ctx(handlerCtx).Err(err).Str("event_id", item.EventId).Msg("failed to save event")
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			}
		}
		return response, nil
	})
}
