package main

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/web"
	brokerinbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = logging.NewDefaultLogger(ctx, logging.WithService("journal"))
	log.Ctx(ctx).Info().Msg("starting application")
	err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config")
	}
	// Create Db Connection
	psqlDb := postgres.SetupPostgresDatabase()
	err = brokerpostgres.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox table")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading AWS config")
	}
	sqsSubscriber, err := sqs.NewSubscriber(ctx, awsCfg, sqs.JournalSqsQueue)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("Error creating SQS subscriber")
	}
	repo := postgres.NewEventRepository(psqlDb)
	eventHandler := inbox.NewEventHandler(sqsSubscriber, repo)
	worker := brokerinbox.NewWorker(eventHandler, 5)
	worker.Start(ctx)
	defer worker.Stop()
	defer psqlDb.Close()
	router := web.NewRouter(repo)
	if err := router.StartRouter(config.APIPort()); err != nil {
		log.Fatal().Err(err).Msg("router stopped")
	}

	log.Ctx(ctx).Info().Msg("application stopped")
}
