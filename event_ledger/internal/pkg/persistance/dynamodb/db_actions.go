package dynamodb

import (
	"context"

	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type DynamoDBClient interface {
	PutItem(ctx context.Context, params *awsdynamo.PutItemInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.PutItemOutput, error)
	GetItem(ctx context.Context, params *awsdynamo.GetItemInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.GetItemOutput, error)
	Query(ctx context.Context, params *awsdynamo.QueryInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.QueryOutput, error)
	Scan(ctx context.Context, params *awsdynamo.ScanInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.ScanOutput, error)
	CreateTable(ctx context.Context, params *awsdynamo.CreateTableInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.CreateTableOutput, error)
	DescribeTable(ctx context.Context, params *awsdynamo.DescribeTableInput, optFns ...func(*awsdynamo.Options)) (*awsdynamo.DescribeTableOutput, error)
}
