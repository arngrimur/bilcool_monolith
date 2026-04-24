package main

import (
	"context"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail/brevo"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/web"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("authentication-http"))
	log.Ctx(ctx).Info().Msg("starting authentication http lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	wauthn, err := webauthn.New(&webauthn.Config{
		RPDisplayName: config.WebAuthnDisplayName(),
		RPID:          config.WebAuthnRPID(),
		RPOrigins:     config.WebAuthnRPOrigins(),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("error creating webauthn instance")
	}

	dbUrl, err := url.Parse(config.DatabaseUrl())
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing database url")
	}
	db := postgresql.SetupPostgresDatabase()
	if err := coutbox.CreateTable(dbUrl); err != nil {
		log.Fatal().Err(err).Msg("error creating outbox table")
	}
	repo := postgresql.NewUsersRepository(db)
	mailSender := brevo.NewSender()
	app := application.New(repo, mailSender, wauthn, config.JWTSecret())
	ginLambda := ginadapter.NewV2(web.NewRouter(app, app, config.JWTSecret()).Engine())

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return ginLambda.ProxyWithContext(ctx, req)
	})
}
