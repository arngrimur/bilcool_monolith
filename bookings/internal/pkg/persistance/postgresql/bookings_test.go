//go:build integrationtest

package postgresql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	_ "github.com/lib/pq"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/migrations"
)

type bookingsTestSuite struct {
	suite.Suite
	// region variables
	testdb.SuiteDbIntegration
	bookingRef uuid.UUID
	now        time.Time
	userRef    uuid.UUID

	// endregion variables
}

// region setup
func (suite *bookingsTestSuite) SetupSuite() {
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.BookingsConnUrlTemplate, migrations.FS)
	suite.bookingRef = uuid.New()
	suite.userRef = uuid.New()
	loc, _ := time.LoadLocation("Etc/UTC")
	suite.now = time.Now().In(loc)
}

func (suite *bookingsTestSuite) TearDownSuite() {
	go suite.CancelFunc()
	_ = suite.Db.Close()
	testcontainers.CleanupContainer(suite.T(), suite.PostgresContainer)
}

func (suite *bookingsTestSuite) BeforeTest(suiteName, testName string) {
	q := "INSERT INTO bookings (booking_reference, start_date, end_date, user_ref) VALUES ($1, $2, $3, $4)"
	_, err := suite.Db.Exec(q, suite.bookingRef, suite.now, suite.now, suite.userRef)
	suite.Require().NoError(err)
	_, err = suite.Db.Exec(q, uuid.New(), suite.now, suite.now, uuid.New())
	suite.Require().NoError(err)
}

func (suite *bookingsTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE bookings CASCADE")
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

// endregion setup
// region tests
func (suite *bookingsTestSuite) TestGetBooking() {
	database := NewBookingsRepository(suite.Db)
	booking, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(suite.userRef, booking.UserRef, "User reference should be the same")
	suite.Require().Equal(suite.now.Truncate(time.Millisecond), booking.StartDate.Truncate(time.Millisecond), "Start date should be the same")

	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(suite.now.Truncate(time.Millisecond), booking.EndDate.Truncate(time.Millisecond), "should be nil")
}

func (suite *bookingsTestSuite) TestFindAllBookings() {
	database := NewBookingsRepository(suite.Db)
	bookings, _ := database.FindAll(context.Background())
	suite.Require().Len(bookings, 2, "Should return 2 bookings")
	suite.Require().NotEqual(bookings[0].BookingReference, bookings[1].BookingReference, "Should return 2 different bookings")
}

func (suite *bookingsTestSuite) TestUpdateExistingBooking() {
	database := NewBookingsRepository(suite.Db)
	booking, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	err := database.UpdateBooking(context.Background(), domain.UpdateBookingRequest{
		BookingReference: suite.bookingRef,
		StartDate:        suite.now.Add(time.Hour),
		EndDate:          suite.now.Add(time.Hour),
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
	_, err := database.Find(context.Background(), domain.BookingRequest{BookingReference: bookingRef})
	suite.Require().Error(err)

	err = database.UpdateBooking(context.Background(), domain.UpdateBookingRequest{
		BookingReference: bookingRef,
		StartDate:        suite.now.Add(time.Hour).Truncate(time.Second),
		EndDate:          suite.now.Add(time.Hour).Truncate(time.Second),
		UserRef:          userRef,
	})
	suite.Require().NoError(err)

	booking2, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: bookingRef})
	suite.Require().Equal(booking2.BookingReference, bookingRef, "Booking reference should be the same")
	suite.Require().WithinDuration(booking2.StartDate, suite.now, time.Hour+time.Minute, "Start date should be the same")
	suite.Require().WithinDuration(booking2.EndDate, suite.now, time.Hour+time.Minute, "Start date should be the same")
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

func (suite *bookingsTestSuite) TestBookingCanBeUpdated() {
	database := NewBookingsRepository(suite.Db)
	booking1 := domain.UpdateBookingRequest{
		UserRef:          uuid.New(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 3, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 5, 0, 0, 0, time.UTC),
	}
	booking2 := domain.UpdateBookingRequest{
		UserRef:          uuid.New(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 5, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 7, 0, 0, 0, time.UTC),
	}
	booking3 := domain.UpdateBookingRequest{
		UserRef:          uuid.New(),
		BookingReference: uuid.New(),
		StartDate:        time.Date(2026, 2, 28, 7, 0, 0, 0, time.UTC),
		EndDate:          time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC),
	}
	ctx := context.Background()
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

// endregion tests
