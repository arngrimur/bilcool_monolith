//go:build integration

package application

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
	"bilcool_monolith/internal/pkg/booking/internal/pkg/persistance/postgresql"
	"bilcool_monolith/internal/pkg/booking/migrations"
	"bilcool_monolith/internal/pkg/testing/testdb"
)

type applicationTestSuite struct {
	suite.Suite
	// region variables
	testdb.SuiteDbIntegration
	bookingRef uuid.UUID
	now        time.Time
	userRef    uuid.UUID
	//endregion variables
}

// region setup
func (suite *applicationTestSuite) SetupSuite() {
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.BookingsConnUrlTemplate, migrations.FS)
	suite.bookingRef = uuid.New()
	suite.userRef = uuid.New()
	loc, _ := time.LoadLocation("Etc/UTC")
	suite.now = time.Now().In(loc)
}
func (suite *applicationTestSuite) TearDownSuite() {
	go suite.CancelFunc()
	testcontainers.CleanupContainer(suite.T(), suite.PostgresContainer)
}
func (suite *applicationTestSuite) BeforeTest(suiteName, testName string) {
	q := "INSERT INTO bookings (booking_reference, start_date, end_date, user_ref) VALUES ($1, $2, $3, $4)"
	_, err := suite.Db.Exec(q, suite.bookingRef, suite.now, suite.now, suite.userRef)
	suite.Require().NoError(err)
}
func (suite *applicationTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE bookings CASCADE")
	suite.Require().NoError(err)
}
func (suite *applicationTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuiteApplication(t *testing.T) {
	suite.Run(t, new(applicationTestSuite))
}

// endregion setup
// region tests
func (suite *applicationTestSuite) TestGetASingleBooking() {
	bookingRepository := postgresql.NewBookingsRepository(suite.Db)
	application := New(bookingRepository)
	booking, err := application.GetBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
}

// endregion tests
