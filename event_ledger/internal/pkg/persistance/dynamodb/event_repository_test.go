//go:build integration

package dynamodb

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/event_ledger/internal/pkg/domain"
	testaws "github.com/arngrimur/bilcool_monolith/testing/aws"
)

const testTableName = "events_test"

type eventRepositoryTestSuite struct {
	suite.Suite
	cloud *testaws.AwsLocalCloud
	repo  *EventRepository
}

func (suite *eventRepositoryTestSuite) SetupSuite() {
	suite.cloud = testaws.SetupLocalCloud(suite.T(), "dynamodb")
	awsCfg := suite.cloud.CreateConfig(suite.T())
	client := NewClientFromConfig(awsCfg)
	err := EnsureTable(suite.cloud.Ctx, client, testTableName)
	suite.Require().NoError(err)
	suite.repo = NewEventRepository(client, testTableName)
}

func (suite *eventRepositoryTestSuite) TearDownSuite() {
	suite.cloud.TearDown(suite.T())
}

func (suite *eventRepositoryTestSuite) makeEvent(producer, eventType string, emittedAt time.Time) domain.EventItem {
	return domain.EventItem{
		EventId:       uuid.New().String(),
		EventType:     eventType,
		CorrelationId: uuid.New().String(),
		Producer:      producer,
		EmittedAt:     emittedAt.UTC().Format(time.RFC3339Nano),
		Payload:       `{"key":"value"}`,
		ReceivedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (suite *eventRepositoryTestSuite) TestSaveEvent() {
	event := suite.makeEvent("bookings", "booking_ended", time.Now())
	err := suite.repo.SaveEvent(suite.cloud.Ctx, event)
	suite.Require().NoError(err)
}

func (suite *eventRepositoryTestSuite) TestSaveEventIdempotent() {
	event := suite.makeEvent("bookings", "booking_ended", time.Now())
	suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, event))
	suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, event))
}

func (suite *eventRepositoryTestSuite) TestQueryByEventId() {
	event := suite.makeEvent("bookings", "booking_ended", time.Now())
	suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, event))

	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		EventId: &event.EventId,
		Limit:   10,
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 1)
	suite.Equal(event.EventId, results[0].EventId)
}

func (suite *eventRepositoryTestSuite) TestQueryByProducer() {
	producer := fmt.Sprintf("test_producer_%s", uuid.New().String())
	now := time.Now()
	for i := 0; i < 3; i++ {
		e := suite.makeEvent(producer, "booking_ended", now.Add(time.Duration(i)*time.Second))
		suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, e))
	}

	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		Producer: &producer,
		Limit:    10,
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 3)
}

func (suite *eventRepositoryTestSuite) TestQueryByEventType() {
	eventType := fmt.Sprintf("test_type_%s", uuid.New().String())
	now := time.Now()
	for i := 0; i < 2; i++ {
		e := suite.makeEvent("bookings", eventType, now.Add(time.Duration(i)*time.Second))
		suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, e))
	}

	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		EventType: &eventType,
		Limit:     10,
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 2)
}

func (suite *eventRepositoryTestSuite) TestQueryByProducerWithTimeRange() {
	producer := fmt.Sprintf("ranged_%s", uuid.New().String())
	base := time.Now().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		e := suite.makeEvent(producer, "booking_ended", base.Add(time.Duration(i)*time.Minute))
		suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, e))
	}

	gte := base.Add(time.Minute)
	lte := base.Add(3 * time.Minute)
	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		Producer:     &producer,
		EmittedAtGte: &gte,
		EmittedAtLte: &lte,
		Limit:        10,
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 3)
}

func (suite *eventRepositoryTestSuite) TestQueryDescendingOrder() {
	producer := fmt.Sprintf("ordered_%s", uuid.New().String())
	base := time.Now().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		e := suite.makeEvent(producer, "booking_ended", base.Add(time.Duration(i)*time.Second))
		suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, e))
	}

	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		Producer:       &producer,
		Limit:          10,
		OrderDirection: "desc",
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 3)
	suite.True(strings.Compare(results[0].EmittedAt, results[1].EmittedAt) >= 0)
}

func (suite *eventRepositoryTestSuite) TestQueryWithOffset() {
	producer := fmt.Sprintf("offset_%s", uuid.New().String())
	base := time.Now().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		e := suite.makeEvent(producer, "booking_ended", base.Add(time.Duration(i)*time.Second))
		suite.Require().NoError(suite.repo.SaveEvent(suite.cloud.Ctx, e))
	}

	results, err := suite.repo.QueryEvents(suite.cloud.Ctx, domain.QueryParams{
		Producer: &producer,
		Limit:    2,
		Offset:   2,
	})
	suite.Require().NoError(err)
	suite.Require().Len(results, 2)
}

func TestRunEventRepositorySuite(t *testing.T) {
	suite.Run(t, new(eventRepositoryTestSuite))
}
