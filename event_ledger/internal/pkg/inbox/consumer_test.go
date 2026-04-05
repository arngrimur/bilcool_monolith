//go:build integration

package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	aws_sns "github.com/aws/aws-sdk-go-v2/service/sns"
	aws_sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	dynstore "github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/persistance/dynamodb"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	broker "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	testaws "github.com/arngrimur/bilcool_monolith/testing/aws"
)

const testTableName = "events_consumer_test"

type consumerTestSuite struct {
	suite.Suite
	cloud         *testaws.AwsLocalCloud
	sqsSubscriber *sqs.SqsSubscriber
	consumer      *Consumer
	snsClient     *aws_sns.Client
	topic         *aws_sns.CreateTopicOutput
	repo          *dynstore.EventRepository
}

func (suite *consumerTestSuite) SetupSuite() {
	suite.cloud = testaws.SetupLocalCloud(suite.T(), "sqs,sns,dynamodb")
	awsCfg := suite.cloud.CreateConfig(suite.T())

	sqsClient := aws_sqs.NewFromConfig(awsCfg)
	q, err := sqsClient.CreateQueue(suite.cloud.Ctx, &aws_sqs.CreateQueueInput{
		QueueName: strPtr("event_ledger_test"),
		Attributes: map[string]string{
			"VisibilityTimeout":             "10",
			"ReceiveMessageWaitTimeSeconds": "1",
		},
	})
	suite.Require().NoError(err)

	suite.snsClient = aws_sns.NewFromConfig(awsCfg)
	suite.topic, err = suite.snsClient.CreateTopic(suite.cloud.Ctx, &aws_sns.CreateTopicInput{
		Name:       strPtr("test_topic"),
		Attributes: map[string]string{"DisplayName": "test_topic", "FifoEndpointResolver": "false"},
	})
	suite.Require().NoError(err)

	attrs, err := sqsClient.GetQueueAttributes(suite.cloud.Ctx, &aws_sqs.GetQueueAttributesInput{
		QueueUrl:       q.QueueUrl,
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
	})
	suite.Require().NoError(err)

	_, err = suite.snsClient.Subscribe(suite.cloud.Ctx, &aws_sns.SubscribeInput{
		Protocol: strPtr("sqs"),
		TopicArn: suite.topic.TopicArn,
		Endpoint: strPtr(attrs.Attributes["QueueArn"]),
	})
	suite.Require().NoError(err)

	sqsSubscriber, err := sqs.NewSubscriber(context.Background(), awsCfg, "event_ledger_test")
	suite.Require().NoError(err)
	suite.sqsSubscriber = sqsSubscriber

	dynamoClient := dynstore.NewClientFromConfig(awsCfg)
	suite.Require().NoError(dynstore.EnsureTable(suite.cloud.Ctx, dynamoClient, testTableName))
	suite.repo = dynstore.NewEventRepository(dynamoClient, testTableName)

	suite.consumer = NewConsumer(sqsSubscriber, 5, suite.repo)
}

func (suite *consumerTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}

func (suite *consumerTestSuite) AfterTest(_, _ string) {
	suite.consumer.Stop()
}

func (suite *consumerTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	if !stats.Passed() {
		buf := strings.Builder{}
		for _, info := range stats.TestStats {
			if !info.Passed {
				buf.WriteString(fmt.Sprintf("Failed %s.%s\n", suiteName, info.TestName))
			}
		}
		suite.Fail(buf.String())
	}
}

func (suite *consumerTestSuite) TestConsumeMessages() {
	producer := fmt.Sprintf("bookings_%s", uuid.New().String())
	for i := 0; i < 5; i++ {
		suite.publishEvent(producer)
	}

	suite.consumer.Start(context.Background())

	suite.Require().Eventuallyf(func() bool {
		results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
			Producer: &producer,
			Limit:    10,
		})
		if err != nil {
			return false
		}
		return len(results) == 5
	}, 10*time.Second, 100*time.Millisecond, "expected 5 events to be stored")
}

func (suite *consumerTestSuite) TestDuplicateEventsAreIdempotent() {
	producer := fmt.Sprintf("dup_%s", uuid.New().String())
	id := uuid.New()
	for i := 0; i < 3; i++ {
		suite.publishEventWithId(producer, &id)
	}

	suite.consumer.Start(context.Background())

	suite.Require().Eventuallyf(func() bool {
		results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
			Producer: &producer,
			Limit:    10,
		})
		if err != nil {
			return false
		}
		return len(results) == 1
	}, 10*time.Second, 100*time.Millisecond, "expected exactly 1 deduplicated event")
}

func (suite *consumerTestSuite) publishEvent(producer string) {
	suite.publishEventWithId(producer, nil)
}

func (suite *consumerTestSuite) publishEventWithId(producer string, id *uuid.UUID) {
	now := time.Now()
	eventId := uuid.New()
	if id != nil {
		eventId = *id
	}
	event := broker.Event{
		EventId:       eventId,
		Type:          "booking_ended",
		CorrelationId: uuid.New(),
		Producer:      producer,
		EmittedAt:     &now,
		Payload:       json.RawMessage(`{"test":true}`),
	}
	msg, err := json.Marshal(event)
	suite.Require().NoError(err)

	_, err = suite.snsClient.Publish(context.Background(), &aws_sns.PublishInput{
		Message:  strPtr(string(msg)),
		TopicArn: suite.topic.TopicArn,
	})
	suite.Require().NoError(err)
}

func TestRunConsumerSuite(t *testing.T) {
	suite.Run(t, new(consumerTestSuite))
}

func strPtr(s string) *string { return &s }
