package sns

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type Publisher interface {
	SendMessage(ctx context.Context, event postgres.Event, topic string) (*awssns.PublishOutput, error)
	SendBatchMessages(ctx context.Context, events []postgres.Event, topic string) (*awssns.PublishBatchOutput, error)
}

//go:generate mockgen -source=publisher.go -destination=publisher_mock.go -package=sns github.com/arngrimur/bilcool_monolith/message_broker
type SnsClientAPI interface {
	Publish(ctx context.Context, params *awssns.PublishInput, optFns ...func(*awssns.Options)) (*awssns.PublishOutput, error)
	PublishBatch(ctx context.Context, params *awssns.PublishBatchInput, optFns ...func(*awssns.Options)) (*awssns.PublishBatchOutput, error)
	ListTopics(ctx context.Context, params *awssns.ListTopicsInput, optFns ...func(*awssns.Options)) (*awssns.ListTopicsOutput, error)
	CreateTopic(ctx context.Context, params *awssns.CreateTopicInput, optFns ...func(*awssns.Options)) (*awssns.CreateTopicOutput, error)
}

type SnsPublisher struct {
	snsClient SnsClientAPI
	cache     *TopicCache
}

type batchOutput struct {
	*awssns.PublishBatchOutput
	err error
}

func NewPublisher(ctx context.Context, cfg aws.Config) (*SnsPublisher, error) {
	snsClient := awssns.NewFromConfig(cfg)
	cache, err := CreateCache(ctx, snsClient)
	if err != nil {
		return nil, err
	}
	return &SnsPublisher{
		snsClient: snsClient,
		cache:     cache,
	}, nil
}

func (s *SnsPublisher) SendMessage(ctx context.Context, event postgres.Event, topic string) (*awssns.PublishOutput, error) {
	if event.EmittedAt == nil {
		event.EmittedAt = new(time.Now())
	}

	topicArn, err := s.cache.GetTopicArn(ctx, topic)
	if err != nil {
		return nil, err
	}

	message, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	messageId, err := s.snsClient.Publish(ctx, &awssns.PublishInput{
		Message:  new(string(message)),
		TopicArn: topicArn,
		Subject:  new(event.Type),
	})
	if err != nil {
		return nil, err
	}
	return messageId, nil
}

func (s *SnsPublisher) SendBatchMessages(ctx context.Context, events []postgres.Event, topic string) (*awssns.PublishBatchOutput, error) {
	waitGroup := sync.WaitGroup{}
	res := &awssns.PublishBatchOutput{}
	const maxBatchSize = 10
	resultChannel := make(chan batchOutput, (len(events)/maxBatchSize)+1)
	topicArn, err := s.cache.GetTopicArn(ctx, topic)
	if err != nil {
		return nil, err
	}

	if len(events) > maxBatchSize {
		for i := 0; i < len(events); i += maxBatchSize {
			if i+maxBatchSize < len(events) {
				waitGroup.Go(func() {
					i := i
					s.sendBatchMessages(ctx, events[i:i+maxBatchSize], *topicArn, resultChannel)
				})
			} else {
				waitGroup.Go(func() {
					i := i
					s.sendBatchMessages(ctx, events[i:], *topicArn, resultChannel)
				})
			}
		}
	} else {
		waitGroup.Go(func() {
			s.sendBatchMessages(ctx, events, *topicArn, resultChannel)
		})
	}
	go func() {
		waitGroup.Wait()
		close(resultChannel)
	}()
	for r := range resultChannel {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		err = errors.Join(err, r.err)
		res.Successful = append(res.Successful, r.PublishBatchOutput.Successful...)
		res.Failed = append(res.Failed, r.PublishBatchOutput.Failed...)
	}
	return res, err
}

func (s *SnsPublisher) sendBatchMessages(ctx context.Context, events []postgres.Event, topicArn string, resultCh chan batchOutput) {
	input := make([]types.PublishBatchRequestEntry, 0)
	totalFailed := make([]types.BatchResultErrorEntry, 0)
	for _, event := range events {
		msg, err := json.Marshal(event)
		if err != nil {
			totalFailed = append(totalFailed, types.BatchResultErrorEntry{
				SenderFault: true,
				Message:     new(err.Error()),
				Id:          new(event.EventId.String()),
			})
			continue
		}
		input = append(input, types.PublishBatchRequestEntry{
			Id:      new(event.EventId.String()),
			Message: new(string(msg)),
		})
	}
	if ctx.Err() != nil {
		result := &awssns.PublishBatchOutput{
			Failed:     totalFailed,
			Successful: make([]types.PublishBatchResultEntry, 0),
		}
		resultCh <- batchOutput{result, ctx.Err()}
		return
	}
	result, err := s.snsClient.PublishBatch(ctx, &awssns.PublishBatchInput{
		PublishBatchRequestEntries: input,
		TopicArn:                   &topicArn,
	})
	if err != nil {
		totalFailed = append(totalFailed, types.BatchResultErrorEntry{
			SenderFault: false,
			Message:     new(err.Error()),
		})
	}
	if result == nil {
		result = &awssns.PublishBatchOutput{
			Failed:     make([]types.BatchResultErrorEntry, 0),
			Successful: make([]types.PublishBatchResultEntry, 0),
		}
	}
	result.Failed = append(result.Failed, totalFailed...)
	resultCh <- batchOutput{result, err}
}
