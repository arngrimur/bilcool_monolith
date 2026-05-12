//go:build integration

package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/migrations"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"

	_ "github.com/lib/pq"

	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
)

type bookingsTestSuite struct {
	suite.Suite
	testdb.SuiteDbIntegration
	bookingRef uuid.UUID
	startTime  time.Time
	userRef    uuid.UUID
	userID     int
	endTime    time.Time
}

func (suite *bookingsTestSuite) SetupSuite() {
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.FS, "bookings_test")
	err := brokerpostgres.CreateTable(suite.ConnString)
	suite.Require().NoError(err)
	suite.bookingRef = uuid.New()
	suite.userRef = uuid.New()
	suite.startTime = time.Now().Add(time.Hour).UTC()
	suite.endTime = suite.startTime.Add(time.Hour)
}

func (suite *bookingsTestSuite) TearDownSuite() {
	suite.SuiteDbIntegration.TearDown(suite.T())
}

func (suite *bookingsTestSuite) BeforeTest(suiteName, testName string) {
	err := suite.Db.QueryRow("INSERT INTO users (userref) VALUES ($1) RETURNING id", suite.userRef).Scan(&suite.userID)
	suite.Require().NoError(err)

	var secondUserID int
	err = suite.Db.QueryRow("INSERT INTO users (userref) VALUES ($1) RETURNING id", uuid.New()).Scan(&secondUserID)
	suite.Require().NoError(err)

	q := "INSERT INTO bookings (booking_reference, start_date, end_date, user_ref) VALUES ($1, $2, $3, $4)"
	_, err = suite.Db.Exec(q, suite.bookingRef, suite.startTime, suite.endTime, suite.userID)
	suite.Require().NoError(err)
	_, err = suite.Db.Exec(q, uuid.New(), suite.startTime, suite.endTime, secondUserID)
	suite.Require().NoError(err)
}

func (suite *bookingsTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE users, inbox, outbox CASCADE")
	suite.Require().NoError(err)
}

func (suite *bookingsTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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

func TestRunSuitebookings(t *testing.T) {
	suite.Run(t, new(bookingsTestSuite))
}

func (suite *bookingsTestSuite) TestGetBooking() {
	database := NewBookingsRepository(suite.Db)
	booking, err := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(suite.userRef, booking.UserRef, "User reference should be the same")
	suite.Require().Equal(suite.startTime.Truncate(time.Millisecond), booking.StartDate.Truncate(time.Millisecond), "Start date should be the same")
	suite.Require().Equal(suite.endTime.Truncate(time.Millisecond), booking.EndDate.Truncate(time.Millisecond), "should be nil")
}

func (suite *bookingsTestSuite) TestFindAllBookings() {
	database := NewBookingsRepository(suite.Db)
	bookings, err := database.FindAll(context.Background())
	suite.Require().NoError(err)
	suite.Require().Len(bookings, 2, "Should return 2 bookings")
	suite.Require().NotEqual(bookings[0].BookingReference, bookings[1].BookingReference, "Should return 2 different bookings")
}

func (suite *bookingsTestSuite) TestUpdateExistingBooking() {
	database := NewBookingsRepository(suite.Db)
	booking, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	err := database.UpdateBooking(context.Background(), domain.UpdateBookingRequest{
		BookingReference: suite.bookingRef,
		StartDate:        suite.startTime.Add(time.Hour),
		EndDate:          suite.endTime.Add(time.Hour),
		UserRef:          suite.userRef,
	})
	suite.Require().NoError(err)
	booking2, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().Equal(booking2.BookingReference, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(time.Hour, booking2.StartDate.Sub(booking.StartDate), "Start date should be the same")
	suite.Require().Equal(time.Hour, booking2.EndDate.Sub(booking.EndDate), "Start date should be the same")
}

func (suite *bookingsTestSuite) TestCreateNewBooking() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()
	bookingRef := uuid.New()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	_, err = database.Find(context.Background(), domain.BookingRequest{BookingReference: bookingRef})
	suite.Require().Error(err)

	err = database.UpdateBooking(context.Background(), domain.UpdateBookingRequest{
		BookingReference: bookingRef,
		StartDate:        suite.startTime.Add(4 * time.Hour).Truncate(time.Second),
		EndDate:          suite.endTime.Add(4 * time.Hour).Truncate(time.Second),
		UserRef:          userRef,
	})
	suite.Require().NoError(err)

	booking2, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: bookingRef})
	suite.Require().Equal(booking2.BookingReference, bookingRef, "Booking reference should be the same")
	suite.Require().WithinDuration(booking2.StartDate, suite.startTime, 4*time.Hour+time.Minute, "Start date should be the same")
	suite.Require().WithinDuration(booking2.EndDate, suite.endTime, 4*time.Hour+time.Minute, "Start date should be the same")
}

func (suite *bookingsTestSuite) TestDeleteBooking() {
	database := NewBookingsRepository(suite.Db)
	err := database.DeleteBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	_, err = database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().Error(err)
	err = database.DeleteBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().Error(err)
}

func (suite *bookingsTestSuite) TestDeleteBookingThatHasStarted() {
	ref := uuid.New()
	q := "INSERT INTO bookings (booking_reference, start_date, end_date, user_ref) VALUES ($1, $2, $3, $4)"
	_, err := suite.Db.Exec(q, ref, suite.startTime.Add(-4*time.Hour), suite.endTime, suite.userID)
	suite.Require().NoError(err)

	database := NewBookingsRepository(suite.Db)
	err = database.DeleteBooking(context.Background(), domain.BookingRequest{BookingReference: ref})
	suite.Require().Error(err)
}

func (suite *bookingsTestSuite) TestBookingCanBeUpdated() {
	database := NewBookingsRepository(suite.Db)
	ctx := context.Background()

	addUser := func() uuid.UUID {
		ref := uuid.New()
		err := database.AddUser(ctx, makeUserMessage(ref, uuid.New().String()))
		suite.Require().NoError(err)
		return ref
	}

	booking1 := domain.UpdateBookingRequest{
		UserRef:          addUser(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 3, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 5, 0, 0, 0, time.UTC),
	}
	booking2 := domain.UpdateBookingRequest{
		UserRef:          addUser(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 5, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 7, 0, 0, 0, time.UTC),
	}
	booking3 := domain.UpdateBookingRequest{
		UserRef:          addUser(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 7, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC),
	}

	err := database.UpdateBooking(ctx, booking1)
	suite.Require().NoError(err)
	err = database.UpdateBooking(ctx, booking2)
	suite.Require().NoError(err)
	err = database.UpdateBooking(ctx, booking3)
	suite.Require().NoError(err)

	suite.T().Run("change booking2 start time earlier booking 1 end time", func(t *testing.T) {
		ti := booking2.StartDate
		defer func() { booking2.StartDate = ti }()
		booking2.StartDate = time.Date(2026, 2, 28, 4, 0, 0, 0, time.UTC)
		err := database.UpdateBooking(ctx, booking2)
		suite.Require().Error(err)
	})

	suite.T().Run("change booking2 end time to 15 minutes later", func(t *testing.T) {
		ti := booking2.EndDate
		defer func() { booking2.EndDate = ti }()
		booking2.EndDate = booking2.EndDate.Add(time.Minute * 15)
		err := database.UpdateBooking(ctx, booking2)
		suite.Require().Error(err)
	})

	suite.T().Run("change booking2 start time to 15 minutes later", func(t *testing.T) {
		ti := booking2.StartDate
		defer func() { booking2.StartDate = ti }()
		booking2.StartDate = booking2.StartDate.Add(time.Minute * 15)
		err := database.UpdateBooking(ctx, booking2)
		suite.Require().NoError(err)
	})
}

func (suite *bookingsTestSuite) TestAddUser() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()
	messageID := uuid.New().String()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, messageID))
	suite.Require().NoError(err)

	var found uuid.UUID
	err = suite.Db.QueryRow("SELECT userref FROM users WHERE userref = $1", userRef).Scan(&found)
	suite.Require().NoError(err)
	suite.Require().Equal(userRef, found)

	var storedMsgID string
	err = suite.Db.QueryRow("SELECT message_id FROM inbox WHERE message_id = $1", messageID).Scan(&storedMsgID)
	suite.Require().NoError(err)
	suite.Require().Equal(messageID, storedMsgID)
}

func (suite *bookingsTestSuite) TestAddUserDuplicateMessageID() {
	database := NewBookingsRepository(suite.Db)
	messageID := uuid.New().String()

	err := database.AddUser(context.Background(), makeUserMessage(uuid.New(), messageID))
	suite.Require().NoError(err)

	err = database.AddUser(context.Background(), makeUserMessage(uuid.New(), messageID))
	suite.Require().Nil(err) // we use ON CONFLICT DO NOTHING
}

func (suite *bookingsTestSuite) TestAddUserDuplicateUserRef() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	err = database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().Nil(err) // we use ON CONFLICT DO NOTHING
}

func (suite *bookingsTestSuite) TestDeleteUser() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	err = database.DeleteUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	var deleted bool
	var deletedAt *time.Time
	err = suite.Db.QueryRow("SELECT deleted, deleted_at FROM users WHERE userref = $1", userRef).Scan(&deleted, &deletedAt)
	suite.Require().NoError(err)
	suite.Require().True(deleted)
	suite.Require().NotNil(deletedAt)
}

func (suite *bookingsTestSuite) TestDeleteUserIdempotent() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	err = database.DeleteUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	err = database.DeleteUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	var count int
	err = suite.Db.QueryRow("SELECT COUNT(*) FROM users WHERE userref = $1 AND deleted = true", userRef).Scan(&count)
	suite.Require().NoError(err)
	suite.Require().Equal(1, count)
}

func (suite *bookingsTestSuite) TestDeleteUserDuplicateMessageID() {
	database := NewBookingsRepository(suite.Db)
	userRef := uuid.New()
	messageID := uuid.New().String()

	err := database.AddUser(context.Background(), makeUserMessage(userRef, uuid.New().String()))
	suite.Require().NoError(err)

	err = database.DeleteUser(context.Background(), makeUserMessage(userRef, messageID))
	suite.Require().NoError(err)

	err = database.DeleteUser(context.Background(), makeUserMessage(userRef, messageID))
	suite.Require().Error(err)
}

func (suite *bookingsTestSuite) TestEndBookingWithPosition() {
	database := NewBookingsRepository(suite.Db)
	pos := &extdomain.Position{Lat: 64.128288, Lon: -21.827774}

	err := database.EndBooking(context.Background(), domain.EndBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Distance:       extdomain.Distance{StartDistance: 0, EndDistance: 12500},
		Position:       pos,
	})
	suite.Require().NoError(err)

	var startDist, endDist int
	err = suite.Db.QueryRow(`
		SELECT d.start_distance, d.end_distance
		FROM distances d JOIN bookings b ON d.fk_booking_id = b.id
		WHERE b.booking_reference = $1`, suite.bookingRef).Scan(&startDist, &endDist)
	suite.Require().NoError(err)
	suite.Require().Equal(0, startDist)
	suite.Require().Equal(12500, endDist)

	var lat, lon float64
	err = suite.Db.QueryRow(`
		SELECT p.lat, p.lon
		FROM positions p JOIN bookings b ON p.fk_booking_id = b.id
		WHERE b.booking_reference = $1`, suite.bookingRef).Scan(&lat, &lon)
	suite.Require().NoError(err)
	suite.Require().InDelta(pos.Lat, lat, 0.000001)
	suite.Require().InDelta(pos.Lon, lon, 0.000001)
}

func (suite *bookingsTestSuite) TestEndBookingWithoutPosition() {
	database := NewBookingsRepository(suite.Db)

	err := database.EndBooking(context.Background(), domain.EndBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Distance:       extdomain.Distance{StartDistance: 0, EndDistance: 8000},
	})
	suite.Require().NoError(err)

	var count int
	err = suite.Db.QueryRow(`
		SELECT COUNT(*)
		FROM positions p JOIN bookings b ON p.fk_booking_id = b.id
		WHERE b.booking_reference = $1`, suite.bookingRef).Scan(&count)
	suite.Require().NoError(err)
	suite.Require().Equal(0, count)
}

func (suite *bookingsTestSuite) TestPauseBooking() {
	database := NewBookingsRepository(suite.Db)
	pos := extdomain.Position{Lat: 59.334591, Lon: 18.063240}

	err := database.PauseBooking(context.Background(), domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos,
	})
	suite.Require().NoError(err)

	var lat, lon float64
	var resumedAt *time.Time
	err = suite.Db.QueryRow(`
		SELECT bp.lat, bp.lon, bp.resumed_at
		FROM booking_pauses bp
		JOIN bookings b ON bp.fk_booking_id = b.id
		WHERE b.booking_reference = $1 AND bp.resumed_at IS NULL`, suite.bookingRef).Scan(&lat, &lon, &resumedAt)
	suite.Require().NoError(err)
	suite.Require().InDelta(pos.Lat, lat, 0.000001)
	suite.Require().InDelta(pos.Lon, lon, 0.000001)
	suite.Require().Nil(resumedAt)
}

func (suite *bookingsTestSuite) TestPauseBooking_AlreadyPaused() {
	database := NewBookingsRepository(suite.Db)
	pos := extdomain.Position{Lat: 59.334591, Lon: 18.063240}

	err := database.PauseBooking(context.Background(), domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos,
	})
	suite.Require().NoError(err)

	err = database.PauseBooking(context.Background(), domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos,
	})
	suite.Require().ErrorIs(err, domain.ErrBookingAlreadyPaused)
}

func (suite *bookingsTestSuite) TestResumeBooking() {
	database := NewBookingsRepository(suite.Db)
	pos := extdomain.Position{Lat: 59.334591, Lon: 18.063240}

	err := database.PauseBooking(context.Background(), domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos,
	})
	suite.Require().NoError(err)

	resp, err := database.ResumeBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().InDelta(pos.Lat, resp.Position.Lat, 0.000001)
	suite.Require().InDelta(pos.Lon, resp.Position.Lon, 0.000001)

	// resumed_at must now be set — no open pause remains
	var count int
	err = suite.Db.QueryRow(`
		SELECT COUNT(*) FROM booking_pauses bp
		JOIN bookings b ON bp.fk_booking_id = b.id
		WHERE b.booking_reference = $1 AND bp.resumed_at IS NULL`, suite.bookingRef).Scan(&count)
	suite.Require().NoError(err)
	suite.Require().Equal(0, count)
}

func (suite *bookingsTestSuite) TestResumeBooking_NotPaused() {
	database := NewBookingsRepository(suite.Db)

	_, err := database.ResumeBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().ErrorIs(err, domain.ErrBookingNotPaused)
}

func (suite *bookingsTestSuite) TestPauseAndResumeMultipleTimes() {
	database := NewBookingsRepository(suite.Db)
	ctx := context.Background()
	pos1 := extdomain.Position{Lat: 59.334591, Lon: 18.063240}
	pos2 := extdomain.Position{Lat: 59.335000, Lon: 18.064000}

	err := database.PauseBooking(ctx, domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos1,
	})
	suite.Require().NoError(err)

	resp, err := database.ResumeBooking(ctx, domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().InDelta(pos1.Lat, resp.Position.Lat, 0.000001)

	err = database.PauseBooking(ctx, domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pos2,
	})
	suite.Require().NoError(err)

	resp, err = database.ResumeBooking(ctx, domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().InDelta(pos2.Lat, resp.Position.Lat, 0.000001)
	suite.Require().InDelta(pos2.Lon, resp.Position.Lon, 0.000001)

	// All pauses must be closed
	var count int
	err = suite.Db.QueryRow(`
		SELECT COUNT(*) FROM booking_pauses bp
		JOIN bookings b ON bp.fk_booking_id = b.id
		WHERE b.booking_reference = $1 AND bp.resumed_at IS NULL`, suite.bookingRef).Scan(&count)
	suite.Require().NoError(err)
	suite.Require().Equal(0, count)
}

// TestEndBookingAfterPauseResume_TotalDistance50km covers a trip where the
// vehicle moves the entire time but the booking is paused for a 6-minute
// segment in the middle:
//
//	30 min active  @ 50 km/h → 25,000 m
//	 6 min paused  @ 50 km/h →  5,000 m  (odometer keeps running)
//	24 min active  @ 50 km/h → 20,000 m
//	                           ────────
//	total odometer delta       50,000 m  (50 km)
//
// The persistence layer must store and return exactly 50,000 m regardless of
// how many pause/resume cycles occurred during the trip.
func (suite *bookingsTestSuite) TestEndBookingAfterPauseResume_TotalDistance50km() {
	database := NewBookingsRepository(suite.Db)
	ctx := context.Background()

	pausePos := extdomain.Position{Lat: 59.334591, Lon: 18.063240} // position at ~25 km mark

	err := database.PauseBooking(ctx, domain.PauseBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Position:       pausePos,
	})
	suite.Require().NoError(err)

	// 6 minutes of driving at 50 km/h while paused — odometer advances 5,000 m
	resp, err := database.ResumeBooking(ctx, domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().InDelta(pausePos.Lat, resp.Position.Lat, 0.000001)
	suite.Require().InDelta(pausePos.Lon, resp.Position.Lon, 0.000001)

	endPos := &extdomain.Position{Lat: 59.450000, Lon: 18.120000}
	err = database.EndBooking(ctx, domain.EndBookingRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Distance:       extdomain.Distance{StartDistance: 0, EndDistance: 50_000},
		Position:       endPos,
	})
	suite.Require().NoError(err)

	var startDist, endDist int
	err = suite.Db.QueryRow(`
		SELECT d.start_distance, d.end_distance
		FROM distances d JOIN bookings b ON d.fk_booking_id = b.id
		WHERE b.booking_reference = $1`, suite.bookingRef).Scan(&startDist, &endDist)
	suite.Require().NoError(err)
	suite.Require().Equal(0, startDist)
	suite.Require().Equal(50_000, endDist)

	dist := extdomain.Distance{StartDistance: startDist, EndDistance: endDist}
	suite.Require().Equal(50_000, dist.Distance(), "total driven distance must be 50 km (50,000 m)")
}

func makeUserMessage(userRef uuid.UUID, messageID string) brokerpostgres.Message {
	payload, _ := json.Marshal(authdomain.UserResponse{UserRef: userRef})
	return brokerpostgres.Message{
		MessageBody: brokerpostgres.MessageBody{
			Message: brokerpostgres.Event{
				EventId: uuid.MustParse(messageID),
				Payload: payload,
			},
		},
	}
}
