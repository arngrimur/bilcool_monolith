package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/application"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance/postgresql"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/web"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("bookings-http"))
	log.Ctx(ctx).Info().Msg("starting bookings http lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	db := postgresql.SetupPostgresDatabase()
	repo := postgresql.NewBookingsRepository(db)
	app := application.New(repo)
	ginLambda := ginadapter.NewV2(web.NewRouter(app.GetBookingsHandler, app.UpdateBookingsHandler).Engine())

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return ginLambda.ProxyWithContext(ctx, req)
	})
}
