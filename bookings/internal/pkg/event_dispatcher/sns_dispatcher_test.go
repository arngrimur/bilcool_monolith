//go:build integration

package event_dispatcher

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	aws_sns "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/migrations"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	outboxdomain "github.com/arngrimur/bilcool_monolith/message_broker/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/aws"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"
)

type snsDispatcherTestSuite struct {
	suite.Suite
	cloud      *aws.AwsLocalCloud
	dispatcher *SnsDispatcher[*sql.DB]
	db         testdb.SuiteDbIntegration
	snsTopic   string

	// region variables

	//endregion variables
}

// region setup
func (suite *snsDispatcherTestSuite) SetupSuite() {
	suite.cloud = aws.SetupLocalCloud(suite.T(), "sns")
	suite.db = testdb.SetupDatabase(suite.T(), migrations.FS, "bookings")
	err := postgres.CreateTable(suite.db.ConnString)
	suite.Require().NoError(err)
	suite.dispatcher, err = NewSnsDispatcher(context.Background(), suite.db.Db, suite.cloud.CreateConfig(suite.T()))
	suite.Require().NoError(err)

	suite.snsTopic = "test_topic"
	snsClient := aws_sns.NewFromConfig(suite.cloud.CreateConfig(suite.T()))
	snsClient.CreateTopic(context.Background(), &aws_sns.CreateTopicInput{Name: &suite.snsTopic})
}
func (suite *snsDispatcherTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}
func (suite *snsDispatcherTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *snsDispatcherTestSuite) AfterTest(suiteName, testName string)  {}
func (suite *snsDispatcherTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuitesnsDispatcher(t *testing.T) {
	suite.Run(t, new(snsDispatcherTestSuite))
}

// endregion setup
// region tests
func (suite *snsDispatcherTestSuite) TestSendBatchMessages() {
	events := []postgres.Event{}
	eventIds := []string{}
	for i := 0; i < 11; i++ {
		b := extdomain.CompletedBooking{
			Booking: extdomain.BookingResponse{
				UserRef:          uuid.New(),
				BookingReference: uuid.New(),
				StartDate:        time.Now().Add(time.Duration(-(i + 1)) * time.Hour),
				EndDate:          time.Now(),
			},
			Distance: extdomain.Distance{
				StartDistance: 100,
				EndDistance:   200,
			},
		}

		message, err := json.Marshal(b)
		suite.Require().NoError(err)
		e := postgres.Event{
			EventId:       uuid.New(),
			Type:          sns.TypeBookingEnded,
			CorrelationId: uuid.New(),
			Producer:      "bookings", // TODO: Change to env variable
			EmittedAt:     new(time.Now()),
			Payload:       message,
		}
		events = append(events, e)
		eventIds = append(eventIds, e.EventId.String())
	}

	out, err := suite.dispatcher.SendBatchMessages(context.Background(), events, suite.snsTopic)
	suite.Require().NoError(err)

	suite.Require().Equal(11, len(out.Successful))
	suite.Require().Equal(0, len(out.Failed))

	for _, e := range out.Successful {
		suite.Require().Contains(eventIds, *e.Id, "Event id should be in the list of sent events")
	}
}

func (suite *snsDispatcherTestSuite) TestExecuteSendOnlySuccessful() {
	controller := gomock.NewController(suite.T())
	dispatcher, err := NewSnsDispatcher(context.Background(), suite.db.Db, suite.cloud.CreateConfig(suite.T()))
	suite.Require().NoError(err)
	suite.Require().NotNil(dispatcher)

	okEvent := postgres.Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		Payload:       []byte(`{"foo":"bar"}`),
	}
	err = postgres.Insert(context.Background(), suite.db.Db, okEvent)
	suite.Require().NoError(err)
	failedEvent := postgres.Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		Payload:       []byte(`{"foo":"bar"}`),
	}
	err = postgres.Insert(context.Background(), suite.db.Db, failedEvent)
	suite.Require().NoError(err)

	out := aws_sns.PublishBatchOutput{
		Failed:     []types.BatchResultErrorEntry{{Id: new(failedEvent.EventId.String()), SenderFault: false, Message: new("error")}},
		Successful: []types.PublishBatchResultEntry{{Id: new(okEvent.EventId.String())}},
	}
	snsMock := sns.NewMockPublisher(controller)
	snsMock.EXPECT().SendBatchMessages(gomock.Any(), gomock.Any(), gomock.Any()).Return(&out, nil).Times(1)
	dispatcher.Publisher = snsMock

	err = dispatcher.Execute(context.Background(), outboxdomain.Table{TableName: "outbox"})
	suite.Require().NoError(err)
	cnt := -1
	q := "select count(*) from outbox where emitted_at IS NULL"
	suite.Require().NoError(suite.db.Db.QueryRow(q).Scan(&cnt))
	suite.Require().Equal(1, cnt, "Should have 1 event left in the outbox")

	cnt = -1
	q = "select count(*) from outbox where emitted_at IS NULL"
	suite.Require().NoError(suite.db.Db.QueryRow(q).Scan(&cnt))
	suite.Require().Equal(1, cnt, "Should have 1 event left in the outbox")
}

// endregion tests
