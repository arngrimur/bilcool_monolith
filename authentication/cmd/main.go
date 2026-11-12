package main

import (
	"context"
	"net/url"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/event_dispatcher"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail/brevo"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail/logsender"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/web"
	soutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging/provider"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/poller"
	snspublisher "github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logging.NewDefaultLogger(ctx, logging.WithService("authentication"))
	log.Ctx(ctx).Info().Msg("starting application")

	err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	dbUrl, err := url.Parse(config.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing database url")
	}
	psqlDb := postgresql.SetupPostgresDatabase()

	err = coutbox.CreateTable(dbUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating outbox table")
	}

	publisher, err := provider.NewPublisher(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating publisher")
	}
	switch config.OutboxMode() {
	case "polling":
		p := poller.New(psqlDb, publisher, snspublisher.TopicUsers)
		go p.RunLoop(ctx, 15*time.Second)
	default: // "replication"
		dispatcher := event_dispatcher.NewDispatcher(psqlDb, publisher)
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
	}

	var mailSender mail.MailSender = brevo.NewSender()
	if config.MailSender() == config.MailSenderLog {
		log.Warn().Msg("MAIL_SENDER=log: security tokens are written to the log, not mailed (development only)")
		mailSender = logsender.NewSender()
	}

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
	webService := web.NewRouter(app, app, config.JWTSecret())

	err = webService.StartRouter(":8080")
	if err != nil {
		log.Fatal().Err(err).Msg("error starting web service")
	}
}
