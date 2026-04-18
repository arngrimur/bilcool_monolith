package dynamodb

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"
)

func NewClient(ctx context.Context) (DynamoDBClient, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return awsdynamo.NewFromConfig(cfg), nil
}

func NewClientFromConfig(cfg aws.Config) DynamoDBClient {
	return awsdynamo.NewFromConfig(cfg)
}

func EnsureTable(ctx context.Context, client DynamoDBClient, tableName string) error {
	_, err := client.DescribeTable(ctx, &awsdynamo.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		return nil
	}

	var notFound *types.ResourceNotFoundException
	if !errors.As(err, &notFound) {
		return err
	}

	log.Ctx(ctx).Info().Str("table", tableName).Msg("creating DynamoDB table")

	_, err = client.CreateTable(ctx, &awsdynamo.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("event_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("emitted_at"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("producer"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("event_type"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("event_id"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("emitted_at"), KeyType: types.KeyTypeRange},
		},
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("producer-emitted_at-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("producer"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("emitted_at"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
			{
				IndexName: aws.String("event_type-emitted_at-index"),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String("event_type"), KeyType: types.KeyTypeHash},
					{AttributeName: aws.String("emitted_at"), KeyType: types.KeyTypeRange},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		},
	})
	return err
}
