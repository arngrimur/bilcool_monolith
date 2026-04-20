package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	"github.com/arngrimur/bilcool_monolith/journal/internal/pkg/web"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("journal-http"))
	log.Ctx(ctx).Info().Msg("starting journal http lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("error reading config")
	}

	db := postgres.SetupPostgresDatabase()
	repo := postgres.NewEventRepository(db)
	ginLambda := ginadapter.NewV2(web.NewRouter(repo).Engine())

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return ginLambda.ProxyWithContext(ctx, req)
	})
}
