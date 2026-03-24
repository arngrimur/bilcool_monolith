package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/aws"

	aws_sns "github.com/aws/aws-sdk-go-v2/service/sns"
)

type subscriberTestSuite struct {
	suite.Suite
	sqsSubscriber *SqsSubscriber
	cloud         *aws.AwsLocalCloud
	snsClient     *aws_sns.Client
	topic         *aws_sns.CreateTopicOutput
	event         postgres.Event

	// region variables

	//endregion variables
}

// region setup
func (suite *subscriberTestSuite) SetupSuite() {
	var err error
	suite.cloud = aws.SetupLocalCloud(suite.T(), "sns,sqs")
	awsCfg := suite.cloud.CreateConfig(suite.T())

	suite.snsClient = aws_sns.NewFromConfig(awsCfg)
	suite.Require().NotNil(suite.snsClient)
	suite.topic, err = suite.snsClient.CreateTopic(suite.cloud.Ctx, &aws_sns.CreateTopicInput{
		Name:       new("test_topic"),
		Attributes: map[string]string{"DisplayName": "test_topic", "FifoEndpointResolver": "false"},
	})
	suite.Require().NoError(err)

	suite.sqsSubscriber = NewSubscriber(context.Background(), awsCfg)
	suite.Require().NotNil(suite.sqsSubscriber)
	q, err := suite.sqsSubscriber.sqsClient.CreateQueue(suite.cloud.Ctx, &sqs.CreateQueueInput{
		QueueName: new("test_queue"),
		Attributes: map[string]string{
			"VisibilityTimeout":             "1",
			"ReceiveMessageWaitTimeSeconds": "20",
		},
	})
	suite.Require().NoError(err)

	attributes, err := suite.sqsSubscriber.sqsClient.GetQueueAttributes(suite.cloud.Ctx, &sqs.GetQueueAttributesInput{
		QueueUrl:       q.QueueUrl,
		AttributeNames: []types.QueueAttributeName{"QueueArn"},
	})
	suite.Require().NoError(err)
	_, err = suite.snsClient.Subscribe(suite.cloud.Ctx, &aws_sns.SubscribeInput{
		Protocol: new("sqs"),
		TopicArn: suite.topic.TopicArn,
		Endpoint: new(attributes.Attributes["QueueArn"]),
	})
	suite.Require().NoError(err)
}
func (suite *subscriberTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}
func (suite *subscriberTestSuite) BeforeTest(suiteName, testName string) {
	suite.addEvent()
}

func (suite *subscriberTestSuite) addEvent() {
	suite.event = postgres.Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		EmittedAt:     nil,
		Payload:       []byte(`{"some":"json"}`),
	}
	message, err := json.Marshal(suite.event)
	suite.Require().NoError(err)
	// send sns message that gets picked up by queue
	_, err = suite.snsClient.Publish(context.Background(), &aws_sns.PublishInput{
		Message:  new(string(message)),
		TopicArn: suite.topic.TopicArn,
	})
	suite.Require().NoError(err)
}
func (suite *subscriberTestSuite) AfterTest(suiteName, testName string) {
	var (
		n   int   = 1
		err error = nil
	)
	for n > 0 {
		messages, _ := suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
		n, err = suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	}
	if err != nil {
		suite.sqsSubscriber.sqsClient.PurgeQueue(suite.cloud.Ctx, &sqs.PurgeQueueInput{})
	}
}
func (suite *subscriberTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	if !stats.Passed() {
		buf := strings.Builder{}
		for _, information := range stats.TestStats {
			if !information.Passed {
				buf.WriteString(fmt.Sprintf("Failed %s.%s\n", suiteName, information.TestName))
			}
		}
		suite.Fail(buf.String())
	}
}
func TestRunSuitepublisher(t *testing.T) {
	suite.Run(t, new(subscriberTestSuite))
}

// endregion setup

func (suite *subscriberTestSuite) TestReadMessages() {
	messages, err := suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)

	for _, body := range messages {
		suite.Require().Equal(suite.event, body.Message)
	}
}

func (suite *subscriberTestSuite) TestMessagesAreHidden() {
	messages, err := suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)

	// Read messages are hidden for a while
	messages, err = suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 0)

	// Now we can the message again
	time.Sleep(1 * time.Second)
	messages, err = suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)
}

func (suite *subscriberTestSuite) TestDeleteMessage() {
	messages, err := suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)

	time.Sleep(3 * time.Second)
	suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	messages, err = suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 0)
}

func (suite *subscriberTestSuite) TestTooManyDeleteMessages() {
	messages := make(map[string]postgres.MessageBody)
	for i := 0; i < 11; i++ {
		messages[fmt.Sprintf("%d", i)] = postgres.MessageBody{}
	}

	n, err := suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	suite.Require().Error(err)
	suite.Require().Equal(0, n)
}

func (suite *subscriberTestSuite) TestDeleteManyMessages() {
	for i := 0; i < 10; i++ {
		suite.addEvent()
	}
	time.Sleep(3 * time.Second)
	messages, err := suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 10)
	n, err := suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	suite.Require().Equal(10, n)

	messages, err = suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 1)
	n, err = suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	suite.Require().NoError(err)
	suite.Require().Equal(1, n)

	messages, err = suite.sqsSubscriber.RetrieveMessages(context.Background(), "test_queue")
	suite.Require().NoError(err)
	suite.Require().Len(messages, 0)
	_, err = suite.sqsSubscriber.DeleteMessages(context.Background(), messages, "test_queue")
	suite.Require().NoError(err)

}

// endregion tests
