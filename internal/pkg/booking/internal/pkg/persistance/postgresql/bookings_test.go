//go:build integration

package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"bilcool_monolith/internal/pkg/booking/internal/pkg/domain"
	"bilcool_monolith/internal/pkg/booking/migrations"
	"bilcool_monolith/internal/pkg/testing/testdb"

	_ "github.com/lib/pq"
)

type bookingsTestSuite struct {
	suite.Suite
	// region variables
	db                *sql.DB
	postgresContainer *testcontainers.DockerContainer
	bookingRef        uuid.UUID
	now               time.Time
	userRef           uuid.UUID

	// endregion variables
}

// region setup
func (suite *bookingsTestSuite) SetupSuite() {
	ctx := context.Background()
	var err error
	suite.postgresContainer, err = testcontainers.Run(
		ctx, "postgres:18",
		testcontainers.WithExposedPorts("5432/tcp"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		),
		testcontainers.WithName("bookings_test_db"),
		testcontainers.WithEnv(map[string]string{"POSTGRES_PASSWORD": "postgres", "POSTGRES_USER": "postgres", "POSTGRES_DB": "bookings"}),
	)
	suite.Require().NoError(err)
	port, err := suite.postgresContainer.MappedPort(ctx, "5432/tcp")
	suite.Require().NoError(err)
	u, _ := url.Parse("postgres://postgres:postgres@localhost:" + port.Port() + "/bookings?sslmode=disable")

	suite.db, err = sql.Open("postgres", u.String())
	suite.Require().NoError(err)

	dbMate := testdb.NewDBMate(suite.T(), testdb.WithEmbeddedFs(migrations.FS))
	err = dbMate.Migrate(suite.db, u)
	suite.Require().NoError(err)

	suite.bookingRef = uuid.New()
	suite.userRef = uuid.New()
	loc, _ := time.LoadLocation("Etc/UTC")
	suite.now = time.Now().In(loc)
}

func (suite *bookingsTestSuite) TearDownSuite() {
	testcontainers.CleanupContainer(suite.T(), suite.postgresContainer)
}

func (suite *bookingsTestSuite) BeforeTest(suiteName, testName string) {
	q := "INSERT INTO bookings (booking_reference, start_date, user_ref) VALUES ($1, $2, $3)"
	_, err := suite.db.Exec(q, suite.bookingRef, suite.now, suite.userRef)
	suite.Require().NoError(err)
	_, err = suite.db.Exec(q, uuid.New(), suite.now, uuid.New())
	suite.Require().NoError(err)
}

func (suite *bookingsTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.db.Exec("TRUNCATE TABLE bookings CASCADE")
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
	_, err := suite.db.Exec("UPDATE bookings SET end_date = $1 WHERE booking_reference = $2", suite.now, suite.bookingRef)
	suite.Require().NoError(err)

	database := NewBookingsDb(suite.db)
	booking, _ := database.Find(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(suite.userRef, booking.UserRef, "User reference should be the same")
	suite.Require().Equal(suite.now.Truncate(time.Millisecond), booking.StartDate.Truncate(time.Millisecond), "Start date should be the same")

	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
	suite.Require().Equal(suite.now.Truncate(time.Millisecond), booking.EndDate.Truncate(time.Millisecond), "should be nil")
}

// endregion tests
