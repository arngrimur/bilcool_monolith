package main

import (
	"context"
	"database/sql"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/web"
	soutbox "github.com/arngrimur/bilcool_monolith/outbox/pkg/outbox/domain"
	coutbox "github.com/arngrimur/bilcool_monolith/outbox/pkg/outbox/postgres"
	"github.com/arngrimur/bilcool_monolith/outbox/pkg/outbox/sqs"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Info().Msg("starting application")
	// Read Config
	c, err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading config")
	}
	// Create Db Connection
	psqlDb := setupPostgresDatabase(c)
	err = coutbox.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox table")
	}
	dbUrl, err := url.Parse(c.DatabaseUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing database url")
	}

	outbox, err := soutbox.NewOutbox(
		ctx,
		dbUrl,
		soutbox.PgOutputPlugin,
		soutbox.NewCreatePublications("bookings_pub", "bookings", []string{coutbox.OutboxTableName}, map[soutbox.ActionName]soutbox.Action{
			soutbox.ActionCommit: sqs.SqsPublisher{},
		}),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating outbox")
	}
	closer, err := outbox.StartReplication()
	if err != nil {
		log.Fatal().Err(err).Msg("Error starting replication")
	}
	defer close(closer)
	// Create Application
	app := application.New(postgresql.NewBookingsRepository(psqlDb))
	webService := web.NewRouter(app.GetBookingsHandler, app.UpdateBookingsHandler)
	webService.StartRouter(":8080")
}

func setupPostgresDatabase(c config.Config) *sql.DB {
	psqlDb, err := sql.Open("postgres", c.DatabaseUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("error opening database connection")
	}
	maxTries := 10
	for i := 1; i <= maxTries; i++ {
		if err := psqlDb.Ping(); err != nil {
			time.Sleep(1 * time.Second)
			log.Err(err).Msgf("error pinging database, attempt: %d", i)
		} else {
			log.Info().Msg("database connection successful")
			return psqlDb
		}
	}
	log.Fatal().Msgf("Error pinging database, gave up after %d attempts", maxTries)
	return nil
}
