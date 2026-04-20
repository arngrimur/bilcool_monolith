package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	bookingsdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("journal-sqs"))
	log.Ctx(ctx).Info().Msg("starting journal sqs lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	db := postgres.SetupPostgresDatabase()
	repo := postgres.NewEventRepository(db)

	lambda.Start(func(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
		response := events.SQSEventResponse{}
		for _, record := range event.Records {
			var body brokerpostgres.MessageBody
			if err := json.Unmarshal([]byte(record.Body), &body); err != nil {
				log.Ctx(ctx).Err(err).Str("message_id", record.MessageId).Msg("failed to parse message")
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
				continue
			}
			msg := brokerpostgres.Message{ReceiptHandle: record.ReceiptHandle, MessageBody: body}
			var err error
			switch body.Message.Type {
			case authdomain.EventUserCreated:
				err = repo.SaveUserCreated(ctx, msg)
			case authdomain.EventUserDeleted:
				err = repo.SaveUserDeleted(ctx, msg)
			case bookingsdomain.EventBookingEnded:
				err = repo.SaveBookingEnded(ctx, msg)
			default:
				continue
			}
			if err != nil {
				log.Ctx(ctx).Err(err).Str("type", body.Message.Type).Msg("failed to process message")
				response.BatchItemFailures = append(response.BatchItemFailures,
					events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
			}
		}
		return response, nil
	})
}
