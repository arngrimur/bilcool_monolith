package main

import (
	"context"
	"database/sql"
	"net/url"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/event_dispatcher"
	bookinginbox "github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/web"
	soutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Ctx(ctx).Info().Msg("starting application")
	// Read Config
	err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config")
	}
	// Create Db Connection
	psqlDb := postgresql.SetupPostgresDatabase()
	err = coutbox.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox table")
	}
	dbUrl, err := url.Parse(config.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing database url")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading AWS config")
	}
	dispatcher, err := event_dispatcher.NewSnsDispatcher[*sql.DB](ctx, psqlDb, awsCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating SNS dispatcher")
	}
	outbox, err := soutbox.NewOutbox(
		ctx,
		dbUrl,
		soutbox.PgOutputPlugin,
		soutbox.NewCreatePublications(
			"bookings_pub",
			"bookings",
			[]string{coutbox.OutboxTableName},
			map[soutbox.ActionName]soutbox.Action{
				soutbox.ActionCommit: dispatcher,
			},
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox")
	}
	closer, err := outbox.StartReplication(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting replication")
	}
	defer close(closer)
	repo := postgresql.NewBookingsRepository(psqlDb)

	sqsSubscriber, err := sqs.NewSubscriber(ctx, awsCfg, sqs.BookingsSqsQueue)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create SQS subscriber")
	}
	eventHandler := bookinginbox.NewEventHandler(sqsSubscriber, repo)
	worker := inbox.NewWorker(eventHandler, 5)
	worker.Start(ctx)

	app := application.New(repo)
	webService := web.NewRouter(app.GetBookingsHandler, app.UpdateBookingsHandler)
	err = webService.StartRouter(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting web service")
	}
}
