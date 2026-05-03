//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	bookingsdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/journal/internal/migrations"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"

	_ "github.com/lib/pq"
)

type eventsTestSuite struct {
	suite.Suite
	testdb.SuiteDbIntegration
	userRef uuid.UUID
}

func (suite *eventsTestSuite) SetupSuite() {
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.FS, "journal_events_test")
	suite.userRef = uuid.New()
}

func (suite *eventsTestSuite) TearDownSuite() {
	suite.SuiteDbIntegration.TearDown(suite.T())
}

func (suite *eventsTestSuite) BeforeTest(suiteName, testName string) {
	_, err := suite.Db.Exec("INSERT INTO users (user_ref, created_at) VALUES ($1, NOW())", suite.userRef)
	suite.Require().NoError(err)
}

func (suite *eventsTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE inbox, users CASCADE")
	suite.Require().NoError(err)
}

func (suite *eventsTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	if !stats.Passed() {
		buf := strings.Builder{}
		for _, information := range stats.TestStats {
			if !information.Passed {
				fmt.Fprintf(&buf, "Failed %s.%s\n", suiteName, information.TestName)
			}
		}
		suite.Fail(buf.String())
	}
}

func TestRunSuiteEvents(t *testing.T) {
	suite.Run(t, new(eventsTestSuite))
}

func (suite *eventsTestSuite) TestSaveBookingEndedWithPosition() {
	repo := NewEventRepository(suite.Db)
	pos := &bookingsdomain.Position{Lat: 64.128288, Lon: -21.827774}

	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        time.Now().Add(-2 * time.Hour),
			EndDate:          time.Now(),
		},
		Distance: bookingsdomain.Distance{StartDistance: 0, EndDistance: 12500},
		Position: pos,
	}

	err := repo.SaveBookingEnded(context.Background(), makeBookingEndedMessage(uuid.New(), cb))
	suite.Require().NoError(err)

	var eventCount int
	err = suite.Db.QueryRow("SELECT COUNT(*) FROM booking_ended_events").Scan(&eventCount)
	suite.Require().NoError(err)
	suite.Require().Equal(1, eventCount)

	var lat, lon float64
	err = suite.Db.QueryRow(`
		SELECT p.lat, p.lon
		FROM positions p JOIN booking_ended_events b ON p.fk_booking_ended_event_id = b.id`).Scan(&lat, &lon)
	suite.Require().NoError(err)
	suite.Require().InDelta(pos.Lat, lat, 0.000001)
	suite.Require().InDelta(pos.Lon, lon, 0.000001)
}

func (suite *eventsTestSuite) TestSaveBookingEndedWithoutPosition() {
	repo := NewEventRepository(suite.Db)

	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        time.Now().Add(-2 * time.Hour),
			EndDate:          time.Now(),
		},
		Distance: bookingsdomain.Distance{StartDistance: 0, EndDistance: 5000},
	}

	err := repo.SaveBookingEnded(context.Background(), makeBookingEndedMessage(uuid.New(), cb))
	suite.Require().NoError(err)

	var posCount int
	err = suite.Db.QueryRow("SELECT COUNT(*) FROM positions").Scan(&posCount)
	suite.Require().NoError(err)
	suite.Require().Equal(0, posCount)
}

func (suite *eventsTestSuite) TestSaveBookingEndedDuplicateIgnored() {
	repo := NewEventRepository(suite.Db)
	id := uuid.New()

	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        time.Now().Add(-2 * time.Hour),
			EndDate:          time.Now(),
		},
		Distance: bookingsdomain.Distance{StartDistance: 0, EndDistance: 3000},
	}

	msg := makeBookingEndedMessage(id, cb)
	err := repo.SaveBookingEnded(context.Background(), msg)
	suite.Require().NoError(err)

	err = repo.SaveBookingEnded(context.Background(), msg)
	suite.Require().NoError(err)

	var count int
	err = suite.Db.QueryRow("SELECT COUNT(*) FROM booking_ended_events").Scan(&count)
	suite.Require().NoError(err)
	suite.Require().Equal(1, count)
}

func (suite *eventsTestSuite) TestGetFinishedBookingsIncludesPosition() {
	repo := NewEventRepository(suite.Db)
	pos := &bookingsdomain.Position{Lat: 64.128288, Lon: -21.827774}

	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        time.Now().Add(-2 * time.Hour),
			EndDate:          time.Now(),
		},
		Distance: bookingsdomain.Distance{StartDistance: 0, EndDistance: 7500},
		Position: pos,
	}

	err := repo.SaveBookingEnded(context.Background(), makeBookingEndedMessage(uuid.New(), cb))
	suite.Require().NoError(err)

	bookings, err := repo.GetFinishedBookings(context.Background(), FinishedBookingFilter{})
	suite.Require().NoError(err)
	suite.Require().Len(bookings, 1)
	suite.Require().NotNil(bookings[0].Position)
	suite.Require().InDelta(pos.Lat, bookings[0].Position.Lat, 0.000001)
	suite.Require().InDelta(pos.Lon, bookings[0].Position.Lon, 0.000001)
}

func (suite *eventsTestSuite) TestGetFinishedBookingsWithoutPosition() {
	repo := NewEventRepository(suite.Db)

	cb := bookingsdomain.CompletedBooking{
		Booking: bookingsdomain.BookingResponse{
			UserRef:          suite.userRef,
			BookingReference: uuid.New(),
			StartDate:        time.Now().Add(-2 * time.Hour),
			EndDate:          time.Now(),
		},
		Distance: bookingsdomain.Distance{StartDistance: 0, EndDistance: 4000},
	}

	err := repo.SaveBookingEnded(context.Background(), makeBookingEndedMessage(uuid.New(), cb))
	suite.Require().NoError(err)

	bookings, err := repo.GetFinishedBookings(context.Background(), FinishedBookingFilter{})
	suite.Require().NoError(err)
	suite.Require().Len(bookings, 1)
	suite.Require().Nil(bookings[0].Position)
}

func makeBookingEndedMessage(id uuid.UUID, cb bookingsdomain.CompletedBooking) brokerpostgres.Message {
	bytes, _ := json.Marshal(cb)
	now := time.Now()
	return brokerpostgres.Message{
		MessageBody: brokerpostgres.MessageBody{
			Message: brokerpostgres.Event{
				EventId:       id,
				Type:          bookingsdomain.EventBookingEnded,
				CorrelationId: uuid.New(),
				Producer:      "bookings",
				EmittedAt:     &now,
				Payload:       bytes,
			},
		},
	}
}
