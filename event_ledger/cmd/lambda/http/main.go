package main

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool-lib/pkg/logging"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/config"
	dynstore "github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/persistance/dynamodb"
	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/web"
)

func main() {
	ctx := logging.NewDefaultLogger(context.Background(), logging.WithService("event-ledger-http"))
	log.Ctx(ctx).Info().Msg("starting event-ledger http lambda")

	if err := config.Init(); err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load AWS config")
	}

	dynamoClient := dynstore.NewClientFromConfig(awsCfg)
	if err := dynstore.EnsureTable(ctx, dynamoClient, config.DynamoTableName()); err != nil {
		log.Fatal().Err(err).Msg("failed to ensure DynamoDB table")
	}

	repo := dynstore.NewEventRepository(dynamoClient, config.DynamoTableName())
	ginLambda := ginadapter.NewV2(web.NewRouter(repo).Engine())

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return ginLambda.ProxyWithContext(ctx, req)
	})
}
