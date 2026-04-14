//go:build integration

package inbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	dynstore "github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/persistance/dynamodb"
	brokerinbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	testaws "github.com/arngrimur/bilcool_monolith/testing/aws"
)

const testTableName = "events_consumer_test"

type consumerTestSuite struct {
	suite.Suite
	cloud         *testaws.AwsLocalCloud
	sqsSubscriber *sqs.SqsSubscriber
	consumer      *Consumer
	worker        *brokerinbox.Worker
	snsClient     *awssns.Client
	topic         *awssns.CreateTopicOutput
	repo          *dynstore.EventRepository
}

func (suite *consumerTestSuite) SetupSuite() {
	suite.cloud = testaws.SetupLocalCloud(suite.T(), "sqs,sns,dynamodb")
	awsCfg := suite.cloud.CreateConfig(suite.T())

	sqsClient := awssqs.NewFromConfig(awsCfg)
	q, err := sqsClient.CreateQueue(suite.cloud.Ctx, &awssqs.CreateQueueInput{
		QueueName: new("event_ledger_test"),
		Attributes: map[string]string{
			"VisibilityTimeout":             "10",
			"ReceiveMessageWaitTimeSeconds": "1",
		},
	})
	suite.Require().NoError(err)

	suite.snsClient = awssns.NewFromConfig(awsCfg)
	suite.topic, err = suite.snsClient.CreateTopic(suite.cloud.Ctx, &awssns.CreateTopicInput{
		Name:       new("test_topic"),
		Attributes: map[string]string{"DisplayName": "test_topic", "FifoEndpointResolver": "false"},
	})
	suite.Require().NoError(err)

	attrs, err := sqsClient.GetQueueAttributes(suite.cloud.Ctx, &awssqs.GetQueueAttributesInput{
		QueueUrl:       q.QueueUrl,
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
	})
	suite.Require().NoError(err)

	_, err = suite.snsClient.Subscribe(suite.cloud.Ctx, &awssns.SubscribeInput{
		Protocol: new("sqs"),
		TopicArn: suite.topic.TopicArn,
		Endpoint: new(attrs.Attributes["QueueArn"]),
	})
	suite.Require().NoError(err)

	sqsSubscriber, err := sqs.NewSubscriber(context.Background(), awsCfg, "event_ledger_test")
	suite.Require().NoError(err)
	suite.sqsSubscriber = sqsSubscriber

	dynamoClient := dynstore.NewClientFromConfig(awsCfg)
	suite.Require().NoError(dynstore.EnsureTable(suite.cloud.Ctx, dynamoClient, testTableName))
	suite.repo = dynstore.NewEventRepository(dynamoClient, testTableName)

	suite.consumer = NewConsumer(sqsSubscriber, suite.repo)
	suite.worker = brokerinbox.NewWorker(suite.consumer, 5)
}

func (suite *consumerTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}

func (suite *consumerTestSuite) AfterTest(_, _ string) {
	suite.worker.Stop()
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
	producer := fmt.Sprintf("event_ledger_%s", uuid.New().String())
	for i := 0; i < 5; i++ {
		suite.publishEvent(producer)
	}

	suite.worker.Start(context.Background())

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
	producer := fmt.Sprintf("event_ledger_%s", uuid.New().String())
	id := uuid.New()
	fixedTime := time.Now()
	for i := 0; i < 3; i++ {
		suite.publishEventWithId(producer, &id, &fixedTime)
	}

	suite.worker.Start(context.Background())
	results := []domain.EventItem{}
	suite.Require().Eventuallyf(func() bool {
		results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
			Producer: &producer,
			Limit:    10,
		})
		if err != nil {
			return false
		}
		return len(results) == 1
	}, 10*time.Second, 100*time.Millisecond, "expected exactly 1 deduplicated event: got", len(results))
}

func (suite *consumerTestSuite) publishEvent(producer string) {
	suite.publishEventWithId(producer, nil, nil)
}

func (suite *consumerTestSuite) publishEventWithId(producer string, id *uuid.UUID, emittedAt *time.Time) {
	now := time.Now()
	if emittedAt != nil {
		now = *emittedAt
	}
	eventId := uuid.New()
	if id != nil {
		eventId = *id
	}
	event := brokerpostgres.Event{
		EventId:       eventId,
		Type:          "booking_ended",
		CorrelationId: uuid.New(),
		Producer:      producer,
		EmittedAt:     &now,
		Payload:       json.RawMessage(`{"test":true}`),
	}
	msg, err := json.Marshal(event)
	suite.Require().NoError(err)

	_, err = suite.snsClient.Publish(context.Background(), &awssns.PublishInput{
		Message:  new(string(msg)),
		TopicArn: suite.topic.TopicArn,
	})
	suite.Require().NoError(err)
}

func TestRunConsumerSuite(t *testing.T) {
	suite.Run(t, new(consumerTestSuite))
}
