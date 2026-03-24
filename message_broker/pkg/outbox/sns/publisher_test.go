package sns

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	aws_sns "github.com/aws/aws-sdk-go-v2/service/sns"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/aws"
)

// region setup

type snsTestSuite struct {
	suite.Suite

	// region variables
	cloud      *aws.AwsLocalCloud
	publisher  *SnsPublisher
	test_topic string

	//endregion variables
}

func (suite *snsTestSuite) SetupSuite() {
	suite.cloud = aws.SetupLocalCloud(suite.T(), "sns,sqs")
	var err error
	suite.publisher, err = NewPublisher(context.Background(), suite.cloud.CreateConfig(suite.T()))
	suite.Require().NoError(err)
	suite.test_topic = "test_topic"
	topic, err := suite.publisher.snsClient.CreateTopic(context.Background(), &aws_sns.CreateTopicInput{
		Name: new(suite.test_topic),
	})
	suite.Require().NoError(err)
	suite.Require().NotNil(topic)
}
func (suite *snsTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}
func (suite *snsTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *snsTestSuite) AfterTest(suiteName, testName string)  {}
func (suite *snsTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSnsTestSuite(t *testing.T) {
	suite.Run(t, new(snsTestSuite))
}

// endregion setup
// region tests

func (suite *snsTestSuite) TestSendMessage() {

	res, err := suite.publisher.SendMessage(context.Background(), postgres.Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		EmittedAt:     nil,
		Payload:       []byte(`{"some":"json"}`),
	}, "test_topic")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().NotNil(res.MessageId)
}

func (suite *snsTestSuite) TestSendSingleBatchMessages() {
	events := suite.createTestEvents(1)
	res, err := suite.publisher.SendBatchMessages(context.Background(), events, "test_topic")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Successful, 1)
}

func (suite *snsTestSuite) TestSendLargeBatchMessages() {
	events := suite.createTestEvents(11)
	res, err := suite.publisher.SendBatchMessages(context.Background(), events, "test_topic")
	suite.Require().NoError(err)
	suite.Require().NotNil(res)
	suite.Require().Len(res.Successful, 11)
	suite.Require().Len(res.Failed, 0)
}

func (suite *snsTestSuite) createTestEvents(no int) []postgres.Event {
	events := make([]postgres.Event, no)

	for i := 0; i < no; i++ {
		events[i] = postgres.Event{
			EventId:       uuid.New(),
			Type:          "test",
			CorrelationId: uuid.New(),
			Producer:      "test",
			EmittedAt:     nil,
			Payload:       []byte(`{"some":"json"}`),
		}
	}
	return events
}

// endregion tests
