package sqs

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
)

type SqsSubscriber struct {
	sqsClient         *aws_sqs.Client
	visibilityTimeout int32
	waitTimeout       int32
	queueUrl          *aws_sqs.GetQueueUrlOutput
}

func NewSubscriber(ctx context.Context, c aws.Config, queueName string) (*SqsSubscriber, error) {
	s := &SqsSubscriber{
		sqsClient:         aws_sqs.NewFromConfig(c),
		visibilityTimeout: 10,
		waitTimeout:       10,
	}
	url, err := s.sqsClient.GetQueueUrl(ctx, &aws_sqs.GetQueueUrlInput{QueueName: &queueName})
	if err != nil {
		return nil, fmt.Errorf("queue %q not found: %w", queueName, err)
	}
	s.queueUrl = url
	return s, nil
}

func (s *SqsSubscriber) RetrieveMessages(ctx context.Context) ([]postgres.Message, error) {
	log.Ctx(ctx).Info().
		Str("queue_url", *s.queueUrl.QueueUrl).
		Msg("Retrieving messages")
	sqsMessages, err := s.sqsClient.ReceiveMessage(ctx, &aws_sqs.ReceiveMessageInput{
		QueueUrl:                    s.queueUrl.QueueUrl,
		MaxNumberOfMessages:         10,
		MessageAttributeNames:       nil,
		MessageSystemAttributeNames: nil,
		ReceiveRequestAttemptId:     nil,
		VisibilityTimeout:           s.visibilityTimeout,
		WaitTimeSeconds:             s.waitTimeout,
	})
	if err != nil {
		return nil, err
	}

	events := make([]postgres.Message, 0)
	for _, message := range sqsMessages.Messages {
		m := postgres.MessageBody{}
		md5Sum := md5.Sum([]byte(*message.Body))
		if fmt.Sprintf("%x", md5Sum) != *message.MD5OfBody {
			return nil, fmt.Errorf("MD5 mismatch")
		}
		err = json.Unmarshal([]byte(*message.Body), &m)
		if err != nil {
			l := log.Ctx(ctx).Error().Err(err)
			if message.Body != nil {
				l = l.Str("body", *message.Body)
			}
			l.Msg("Unmarshal failed")
		}

		events = append(events, postgres.Message{
			ReceiptHandle: *message.ReceiptHandle,
			MessageBody:   m,
		})
	}
	return events, nil
}

// DeleteMessages deletes the messages from the queue
// returns the number of messages deleted and an error if something went wrong
func (s *SqsSubscriber) DeleteMessages(ctx context.Context, messages []postgres.Message) (int, error) {
	if len(messages) > 10 {
		return 0, fmt.Errorf("too many messages to delete, max 10 is allowed")
	}
	if len(messages) == 0 {
		return 0, nil
	}

	receiptHandles := make([]string, 0, len(messages))
	for _, k := range messages {
		receiptHandles = append(receiptHandles, k.ReceiptHandle)
	}
	return s.deleteBatch(ctx, receiptHandles, s.queueUrl)

}

func (s *SqsSubscriber) deleteBatch(ctx context.Context, receiptHandles []string, url *aws_sqs.GetQueueUrlOutput) (int, error) {
	entries := make([]types.DeleteMessageBatchRequestEntry, len(receiptHandles))
	for i, receiptHandle := range receiptHandles {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(fmt.Sprintf("%d", i)),
			ReceiptHandle: aws.String(receiptHandle),
		}
	}
	result, err := s.sqsClient.DeleteMessageBatch(ctx, &aws_sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: url.QueueUrl,
	})
	if err != nil {
		return 0, nil
	}
	return len(result.Successful), nil
}

func (s *SqsSubscriber) VisibilityTimeout() int {
	return int(s.visibilityTimeout)
}
