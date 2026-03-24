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
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/migrations"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
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
	suite.cloud = aws.SetupLocalCloud(suite.T(), "sns", "sns_dispatcher_test")
	suite.db = testdb.SetupDatabase(suite.T(), migrations.BookingsConnUrlTemplate, migrations.FS, "sns_dispatcher_test")
	var err error
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
		b := domain.CompletedBooking{
			Booking: domain.BookingResponse{
				UserRef:          uuid.New(),
				BookingReference: uuid.New(),
				StartDate:        time.Now().Add(time.Duration(-(i + 1)) * time.Hour),
				EndDate:          time.Now(),
			},
			Distance: domain.Distance{
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
	suite.Fail("How can we test this?")
}

// endregion tests
