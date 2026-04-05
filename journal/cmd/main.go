package main

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Ctx(ctx).Info().Msg("starting application")
	err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config")
	}
	// Create Db Connection
	psqlDb := postgres.SetupPostgresDatabase()
	err = coutbox.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox table")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading AWS config")
	}
	sqsSubscriber, err := sqs.NewSubscriber(ctx, awsCfg, "journal")
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("Error creating SQS subscriber")
	}
	consumer := inbox.NewConsumer(sqsSubscriber, 5, psqlDb)
	consumer.Start(ctx)
	for {
		select {
		case <-ctx.Done():
			consumer.Stop()
			_ = psqlDb.Close()
		}
	}
	log.Ctx(ctx).Info().Msg("application stopped")
}
