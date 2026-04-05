package dynamodb

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
)

type EventRepository struct {
	client    DynamoDBClient
	tableName string
}

func NewEventRepository(client DynamoDBClient, tableName string) *EventRepository {
	return &EventRepository{client: client, tableName: tableName}
}

func (r *EventRepository) SaveEvent(ctx context.Context, item domain.EventItem) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return err
	}
	_, err = r.client.PutItem(ctx, &awsdynamo.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(event_id)"),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return nil
		}
		return err
	}
	return nil
}

func (r *EventRepository) QueryEvents(ctx context.Context, p domain.QueryParams) ([]domain.EventItem, error) {
	if p.EventId != nil {
		return r.queryByEventId(ctx, *p.EventId)
	}
	if p.Producer != nil {
		return r.queryGSI(ctx, "producer-emitted_at-index", "producer", *p.Producer, p)
	}
	if p.EventType != nil {
		return r.queryGSI(ctx, "event_type-emitted_at-index", "event_type", *p.EventType, p)
	}
	return r.scan(ctx, p)
}

func (r *EventRepository) queryByEventId(ctx context.Context, eventId string) ([]domain.EventItem, error) {
	out, err := r.client.Query(ctx, &awsdynamo.QueryInput{
		TableName:              aws.String(r.tableName),
		KeyConditionExpression: aws.String("event_id = :eid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":eid": &types.AttributeValueMemberS{Value: eventId},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, err
	}
	return unmarshalItems(out.Items)
}

func (r *EventRepository) queryGSI(ctx context.Context, indexName, pkAttr, pkValue string, p domain.QueryParams) ([]domain.EventItem, error) {
	keyCondParts := []string{pkAttr + " = :pk"}
	exprVals := map[string]types.AttributeValue{
		":pk": &types.AttributeValueMemberS{Value: pkValue},
	}

	if p.EmittedAt != nil {
		keyCondParts = append(keyCondParts, "emitted_at = :ea")
		exprVals[":ea"] = &types.AttributeValueMemberS{Value: p.EmittedAt.UTC().Format(time.RFC3339Nano)}
	} else if p.EmittedAtGte != nil && p.EmittedAtLte != nil {
		keyCondParts = append(keyCondParts, "emitted_at BETWEEN :eagte AND :ealte")
		exprVals[":eagte"] = &types.AttributeValueMemberS{Value: p.EmittedAtGte.UTC().Format(time.RFC3339Nano)}
		exprVals[":ealte"] = &types.AttributeValueMemberS{Value: p.EmittedAtLte.UTC().Format(time.RFC3339Nano)}
	} else if p.EmittedAtGte != nil {
		keyCondParts = append(keyCondParts, "emitted_at >= :eagte")
		exprVals[":eagte"] = &types.AttributeValueMemberS{Value: p.EmittedAtGte.UTC().Format(time.RFC3339Nano)}
	} else if p.EmittedAtLte != nil {
		keyCondParts = append(keyCondParts, "emitted_at <= :ealte")
		exprVals[":ealte"] = &types.AttributeValueMemberS{Value: p.EmittedAtLte.UTC().Format(time.RFC3339Nano)}
	}

	ascending := p.OrderDirection != "desc"
	fetchLimit := int32(p.Offset + p.Limit)

	var collected []domain.EventItem
	var lastKey map[string]types.AttributeValue

	for {
		remaining := int(fetchLimit) - len(collected)
		if remaining <= 0 {
			break
		}
		input := &awsdynamo.QueryInput{
			TableName:                 aws.String(r.tableName),
			IndexName:                 aws.String(indexName),
			KeyConditionExpression:    aws.String(strings.Join(keyCondParts, " AND ")),
			ExpressionAttributeValues: exprVals,
			ScanIndexForward:          aws.Bool(ascending),
			Limit:                     aws.Int32(int32(remaining)),
			ExclusiveStartKey:         lastKey,
		}
		out, err := r.client.Query(ctx, input)
		if err != nil {
			return nil, err
		}
		items, err := unmarshalItems(out.Items)
		if err != nil {
			return nil, err
		}
		collected = append(collected, items...)
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	if p.Offset >= len(collected) {
		return []domain.EventItem{}, nil
	}
	end := p.Offset + p.Limit
	if end > len(collected) {
		end = len(collected)
	}
	return collected[p.Offset:end], nil
}

func (r *EventRepository) scan(ctx context.Context, p domain.QueryParams) ([]domain.EventItem, error) {
	filterParts := make([]string, 0)
	exprVals := map[string]types.AttributeValue{}

	if p.EmittedAt != nil {
		filterParts = append(filterParts, "emitted_at = :ea")
		exprVals[":ea"] = &types.AttributeValueMemberS{Value: p.EmittedAt.UTC().Format(time.RFC3339Nano)}
	} else {
		if p.EmittedAtGte != nil {
			filterParts = append(filterParts, "emitted_at >= :eagte")
			exprVals[":eagte"] = &types.AttributeValueMemberS{Value: p.EmittedAtGte.UTC().Format(time.RFC3339Nano)}
		}
		if p.EmittedAtLte != nil {
			filterParts = append(filterParts, "emitted_at <= :ealte")
			exprVals[":ealte"] = &types.AttributeValueMemberS{Value: p.EmittedAtLte.UTC().Format(time.RFC3339Nano)}
		}
	}

	var collected []domain.EventItem
	var lastKey map[string]types.AttributeValue
	target := p.Offset + p.Limit

	for len(collected) < target {
		input := &awsdynamo.ScanInput{
			TableName:         aws.String(r.tableName),
			ExclusiveStartKey: lastKey,
		}
		if len(filterParts) > 0 {
			input.FilterExpression = aws.String(strings.Join(filterParts, " AND "))
			input.ExpressionAttributeValues = exprVals
		}
		out, err := r.client.Scan(ctx, input)
		if err != nil {
			return nil, err
		}
		items, err := unmarshalItems(out.Items)
		if err != nil {
			return nil, err
		}
		collected = append(collected, items...)
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	if p.Offset >= len(collected) {
		return []domain.EventItem{}, nil
	}
	end := p.Offset + p.Limit
	if end > len(collected) {
		end = len(collected)
	}
	return collected[p.Offset:end], nil
}

func unmarshalItems(items []map[string]types.AttributeValue) ([]domain.EventItem, error) {
	result := make([]domain.EventItem, 0, len(items))
	for _, item := range items {
		var e domain.EventItem
		if err := attributevalue.UnmarshalMap(item, &e); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}
