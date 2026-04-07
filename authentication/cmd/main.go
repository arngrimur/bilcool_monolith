package main

import (
	"context"
	"database/sql"
	"net/url"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/event_dispatcher"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail/brevo"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/web"
	soutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Msg("starting application")

	err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	psqlDb := postgresql.SetupPostgresDatabase()

	err = coutbox.CreateTable(psqlDb)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating outbox table")
	}

	dbUrl, err := url.Parse(config.DatabaseUrl())
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

	mailSender := brevo.NewSender()

	wauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: config.WebAuthnDisplayName(),
		RPID:          config.WebAuthnRPID(),
		RPOrigins:     config.WebAuthnRPOrigins(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error creating webauthn instance")
	}

	repo := postgresql.NewUsersRepository(psqlDb)
	app := application.New(repo, mailSender, wauthn, config.JWTSecret())
	webService := web.NewRouter(app, app)

	err = webService.StartRouter(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("error starting web service")
	}
}
