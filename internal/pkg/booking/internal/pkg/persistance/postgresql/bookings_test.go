//go:build integration

package postgresql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
	"bilcool_monolith/internal/pkg/booking/migrations"
	"bilcool_monolith/internal/pkg/testing/testdb"

	_ "github.com/lib/pq"
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

// endregion tests
