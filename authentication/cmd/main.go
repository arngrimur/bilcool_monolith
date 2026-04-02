package main

import (
	"context"
	"database/sql"
	"net/url"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/event_dispatcher"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail/ses"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/web"
	soutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Msg("starting application")

	c, err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	psqlDb := setupPostgresDatabase(c)

	err = coutbox.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating outbox table")
	}

	dbUrl, err := url.Parse(c.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing database url")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading AWS config")
	}

	dispatcher, err := event_dispatcher.NewSnsDispatcher[*sql.DB](ctx, psqlDb, awsCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating SNS dispatcher")
	}

	outbox, err := soutbox.NewOutbox(
		ctx,
		dbUrl,
		soutbox.PgOutputPlugin,
		soutbox.NewCreatePublications(
			"authentication_pub",
			"authentication",
			[]string{coutbox.OutboxTableName},
			map[soutbox.ActionName]soutbox.Action{
				soutbox.ActionCommit: dispatcher,
			},
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating outbox")
	}

	closer, err := outbox.StartReplication(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error starting replication")
	}
	defer close(closer)

	mailSender := ses.NewSeSender(awsCfg, c.SESFromEmail())

	wauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: c.WebAuthnDisplayName(),
		RPID:          c.WebAuthnRPID(),
		RPOrigins:     c.WebAuthnRPOrigins(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error creating webauthn instance")
	}

	repo := postgresql.NewUsersRepository(psqlDb)
	app := application.New(repo, mailSender, wauthn, c.JWTSecret())
	webService := web.NewRouter(app, app)

	err = webService.StartRouter(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("error starting web service")
	}
}

func setupPostgresDatabase(c config.Config) *sql.DB {
	psqlDb, err := sql.Open("postgres", c.DatabaseUrl())
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
	log.Fatal().Msgf("error pinging database, gave up after %d attempts", maxTries)
	return nil
}
