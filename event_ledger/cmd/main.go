package main

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/inbox"
	dynstore "github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/persistance/dynamodb"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/web"
	brokerinbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Msg("starting event-ledger")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS config")
	}

	dynamoClient := dynstore.NewClientFromConfig(awsCfg)
	if err := dynstore.EnsureTable(ctx, dynamoClient, config.DynamoTableName()); err != nil {
		log.Fatal().Err(err).Msg("failed to ensure DynamoDB table")
	}

	repo := dynstore.NewEventRepository(dynamoClient, config.DynamoTableName())

	sqsSubscriber, err := sqs.NewSubscriber(ctx, awsCfg, sqs.EventLedgerSqsQueue)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create SQS subscriber")
	}

	consumer := inbox.NewConsumer(sqsSubscriber, repo)
	worker := brokerinbox.NewWorker(consumer, 5)
	worker.Start(ctx)
	defer worker.Stop()

	router := web.NewRouter(repo)
	if err := router.StartRouter(config.APIPort()); err != nil {
		log.Fatal().Err(err).Msg("router stopped")
	}

	log.Ctx(ctx).Info().Msg(
		"event-ledger is running",
	)
	log.Info().Msg("stopping event-ledge")
}
