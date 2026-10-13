//go:build integration

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	authdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/migrations"
	brokerpostgres "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"

	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/bookings/internal/pkg/persistance/postgresql"
	extdomain "github.com/arngrimur/bilcool_monolith/bookings/pkg/domain"
)

type noopDistanceCalc struct{}

func (noopDistanceCalc) CalculateRoadDistance(_ context.Context, _ []extdomain.TrackPoint) (int, error) {
	return 0, nil
}

type stubRouteMatcher struct{ lines [][]extdomain.Position }

func (m stubRouteMatcher) MatchRoute(_ context.Context, _ []extdomain.TrackPoint) ([][]extdomain.Position, error) {
	return m.lines, nil
}

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
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.FS, "application_test")
	suite.bookingRef = uuid.New()
	suite.userRef = uuid.New()
	loc, _ := time.LoadLocation("Etc/UTC")
	suite.now = time.Now().In(loc)
}
func (suite *applicationTestSuite) TearDownSuite() {
	go suite.CancelFunc()
	_ = suite.Db.Close()
	testcontainers.CleanupContainer(suite.T(), suite.PostgresContainer)
}
func (suite *applicationTestSuite) BeforeTest(suiteName, testName string) {
	var userID int
	err := suite.Db.QueryRow("INSERT INTO users (userref) VALUES ($1) RETURNING id", suite.userRef).Scan(&userID)
	suite.Require().NoError(err)

	q := "INSERT INTO bookings (booking_reference, start_date, end_date, user_ref) VALUES ($1, $2, $3, $4)"
	_, err = suite.Db.Exec(q, suite.bookingRef, suite.now, suite.now, userID)
	suite.Require().NoError(err)
}
func (suite *applicationTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE users, inbox CASCADE")
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
	application := New(bookingRepository, noopDistanceCalc{}, nil)
	booking, err := application.GetBooking(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.bookingRef, booking.BookingReference, "Booking reference should be the same")
}

func (suite *applicationTestSuite) TestUpdateBooking() {
	bookingRepository := postgresql.NewBookingsRepository(suite.Db)
	application := New(bookingRepository, noopDistanceCalc{}, nil)
	r := domain.UpdateBookingRequest{
		BookingReference: uuid.New(),
		StartDate:        suite.now,
		EndDate:          suite.now,
		UserRef:          uuid.New(),
	}
	payload, _ := json.Marshal(authdomain.UserResponse{UserRef: r.UserRef})
	addMsg := brokerpostgres.Message{
		MessageBody: brokerpostgres.MessageBody{
			Message: brokerpostgres.Event{EventId: uuid.New(), Payload: payload},
		},
	}
	err := bookingRepository.AddUser(context.Background(), addMsg)
	suite.Require().NoError(err)
	err = application.UpdateBooking(context.Background(), r)
	suite.Require().NoError(err)
	e, err := application.GetBooking(context.Background(), domain.BookingRequest{BookingReference: r.BookingReference})
	suite.Require().NoError(err)
	suite.Require().Equal(r.BookingReference, e.BookingReference, "Booking reference should be the same")
	suite.Require().WithinDuration(r.StartDate, e.StartDate, time.Second, "Start date should be the same")
	suite.Require().WithinDuration(r.EndDate, e.EndDate, time.Second, "Start date should be the same")
}


func (suite *applicationTestSuite) TestGetRoute() {
	bookingRepository := postgresql.NewBookingsRepository(suite.Db)
	snapped := [][]extdomain.Position{{{Lat: 59.331, Lon: 18.061}, {Lat: 59.339, Lon: 18.069}}}
	application := New(bookingRepository, noopDistanceCalc{}, stubRouteMatcher{lines: snapped})
	t0 := time.Now().UTC()
	err := application.AddTrackPoints(context.Background(), domain.AddTrackPointsRequest{
		BookingRequest: domain.BookingRequest{BookingReference: suite.bookingRef},
		Points: []extdomain.TrackPoint{
			{Lat: 59.33, Lon: 18.06, RecordedAt: t0},
			{Lat: 59.34, Lon: 18.07, RecordedAt: t0.Add(2 * time.Minute)},
		},
	})
	suite.Require().NoError(err)

	route, err := application.GetRoute(context.Background(), domain.BookingRequest{BookingReference: suite.bookingRef})

	suite.Require().NoError(err)
	suite.Require().Len(route.Points, 2)
	suite.Require().Equal(snapped, route.Route)
}

// endregion tests
