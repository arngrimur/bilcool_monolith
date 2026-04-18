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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"

	bookingsdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/journal/internal/migrations"
	journalpostgres "github.com/arngrimur/bilcool_monolith/journal/internal/pkg/persistance/postgres"
	msinbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/inbox/sqs"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/aws"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"
)

type sqsSubscriberTestSuite struct {
	suite.Suite
	// region variables
	cloud         *aws.AwsLocalCloud
	sqsSubscriber *sqs.SqsSubscriber
	consumer      *msinbox.Worker
	snsClient     *aws_sns.Client
	topic         *aws_sns.CreateTopicOutput
	dbIntegration testdb.SuiteDbIntegration
	userRef       uuid.UUID
	//endregion variables
}

// region setup
func (suite *sqsSubscriberTestSuite) SetupSuite() {
	suite.cloud = aws.SetupLocalCloud(suite.T(), "sns,sqs")
	awsCfg := suite.cloud.CreateConfig(suite.T())

	sqsClient := aws_sqs.NewFromConfig(awsCfg)
	q, err := sqsClient.CreateQueue(suite.cloud.Ctx, &aws_sqs.CreateQueueInput{
		QueueName: new("test_queue"),
		Attributes: map[string]string{
			"VisibilityTimeout":             "1",
			"ReceiveMessageWaitTimeSeconds": "20",
		},
	})
	suite.Require().NoError(err)

	suite.snsClient = aws_sns.NewFromConfig(awsCfg)
	suite.Require().NotNil(suite.snsClient)
	suite.topic, err = suite.snsClient.CreateTopic(suite.cloud.Ctx, &aws_sns.CreateTopicInput{
		Name:       new("test_topic"),
		Attributes: map[string]string{"DisplayName": "test_topic", "FifoEndpointResolver": "false"},
	})
	suite.Require().NoError(err)

	attributes, err := sqsClient.GetQueueAttributes(suite.cloud.Ctx, &aws_sqs.GetQueueAttributesInput{
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

	sqsSubscriber, err := sqs.NewSubscriber(context.Background(), awsCfg, "test_queue")
	suite.Require().NoError(err)
	suite.Require().NotNil(sqsSubscriber)
	suite.dbIntegration = testdb.SetupDatabase(suite.T(), migrations.FS, "journal_test")

	repo := journalpostgres.NewEventRepository(suite.dbIntegration.Db)
	handler := NewEventHandler(sqsSubscriber, repo)
	suite.consumer = msinbox.NewWorker(handler, 10)
	suite.userRef = uuid.New()
	_, err = suite.dbIntegration.Db.Exec("INSERT INTO users(id,user_ref,created_at) VALUES (1,$1,$2 )", suite.userRef.String(), time.Now().Add(time.Hour*24*-31))
	suite.Require().NoError(err)
}
func (suite *sqsSubscriberTestSuite) TearDownSuite() {
	suite.dbIntegration.TearDown(suite.T())
	suite.cloud.TearDown(suite.T())
}
func (suite *sqsSubscriberTestSuite) BeforeTest(suiteName, testName string) {

}

func (suite *sqsSubscriberTestSuite) AfterTest(suiteName, testName string) {
	suite.consumer.Stop()
	zerolog.SetGlobalLevel(zerolog.NoLevel)
}

func (suite *sqsSubscriberTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuitesqsSubscriber(t *testing.T) {
	suite.Run(t, new(sqsSubscriberTestSuite))
}

// endregion setup
// region tests
func (suite *sqsSubscriberTestSuite) TestReadMessages() {
	for i := 0; i < 300; i++ {
		suite.addEvent(nil)
	}
	suite.consumer.Start(context.Background())
	var inboxCount int
	suite.Require().Eventuallyf(func() bool {
		row := suite.dbIntegration.Db.QueryRow(`select count(*) from inbox`)
		if err := row.Scan(&inboxCount); err != nil {
			return false
		}
		return inboxCount == 300
	}, 5*time.Second, time.Millisecond, fmt.Sprintf("failed to read messages from queue. got %d messages, WANTED 300", inboxCount))

	var eventCount int
	suite.Require().Eventuallyf(func() bool {
		row := suite.dbIntegration.Db.QueryRow(`select count(*) from booking_ended_events`)
		if err := row.Scan(&eventCount); err != nil {
			return false
		}
		return eventCount == 300
	}, 5*time.Second, time.Millisecond, fmt.Sprintf("failed to read events from queue. got %d events, WANTED 300", eventCount))
}

func (suite *sqsSubscriberTestSuite) TestDuplicateEventsAreStopped() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	id := uuid.New()
	for i := 0; i < 2; i++ {
		suite.addEvent(&id)
	}
	suite.consumer.Start(context.Background())
	var inboxCount int
	suite.Require().Eventuallyf(func() bool {
		row := suite.dbIntegration.Db.QueryRow(`select count(*) from inbox`)
		if err := row.Scan(&inboxCount); err != nil {
			return false
		}
		return inboxCount == 1
	}, 1*time.Second, time.Millisecond, fmt.Sprintf("failed to read messages from queue. got %d messages, WANTED 300", inboxCount))

	var eventCount int
	suite.Require().Eventuallyf(func() bool {
		row := suite.dbIntegration.Db.QueryRow(`select count(*) from booking_ended_events`)
		if err := row.Scan(&eventCount); err != nil {
			return false
		}
		return eventCount == 1
	}, 1*time.Second, time.Millisecond, fmt.Sprintf("failed to read events from queue. got %d events, WANTED 300", eventCount))
}

// endregion tests

func (suite *sqsSubscriberTestSuite) addEvent(id *uuid.UUID) {
	now := time.Now()
	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        now.Add(time.Hour * -4),
			EndDate:          now,
		},
		Distance: bookingsdomain.Distance{
			StartDistance: 1,
			EndDistance:   200,
		},
	}
	bytes, err := json.Marshal(cb)
	if id == nil {
		id = new(uuid.New())
	}
	event := postgres.Event{
		EventId:       *id,
		Type:          bookingsdomain.EventBookingEnded,
		CorrelationId: uuid.New(),
		Producer:      "bookings",
		EmittedAt:     new(time.Now()),
		Payload:       bytes,
	}
	message, err := json.Marshal(event)
	suite.Require().NoError(err)
	// send sns message that gets picked up by queue
	_, err = suite.snsClient.Publish(context.Background(), &aws_sns.PublishInput{
		Message:  new(string(message)),
		TopicArn: suite.topic.TopicArn,
	})
	suite.Require().NoError(err)
}
