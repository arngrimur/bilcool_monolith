package postgres

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"
)

type postgresTestSuite struct {
	suite.Suite
	dbIntegration testdb.SuiteDbIntegration

	// region variables

	//endregion variables
}

// region setup
func (suite *postgresTestSuite) SetupSuite() {
	suite.dbIntegration = testdb.SetupDatabase(suite.T(), embed.FS{}, OutboxTableName)
	err := CreateTable(suite.dbIntegration.Db)
	suite.Require().NoError(err)
}
func (suite *postgresTestSuite) TearDownSuite() {
	suite.dbIntegration.TearDown(suite.T())
}
func (suite *postgresTestSuite) BeforeTest(suiteName, testName string) {
	event := Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		Payload:       []byte(`{"foo":"bar"}`),
	}
	err := Insert(context.Background(), suite.dbIntegration.Db, event)
	suite.Require().NoError(err)
}
func (suite *postgresTestSuite) AfterTest(suiteName, testName string) {
	suite.dbIntegration.Db.Exec("TRUNCATE TABLE " + OutboxTableName + " CASCADE")
}
func (suite *postgresTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuitepostgres(t *testing.T) {
	suite.Run(t, new(postgresTestSuite))
}

// endregion setup
// region tests
func (suite *postgresTestSuite) TestInsertInvalidEventJson() {
	e := Event{
		EventId:       uuid.New(),
		Type:          "test",
		CorrelationId: uuid.New(),
		Producer:      "test",
		Payload:       []byte(`sadfsafdsd`),
	}
	event := e
	err := Insert(context.Background(), suite.dbIntegration.Db, event)
	suite.Require().Error(err)
}

func (suite *postgresTestSuite) TestFindAll() {
	events, err := FindAllNewEvents(context.Background(), suite.dbIntegration.Db)
	suite.Require().NoError(err)
	suite.Require().Len(events, 1, "Should return 1 events")
}

func (suite *postgresTestSuite) TestMarkEmitted() {
	events, err := FindAllNewEvents(context.Background(), suite.dbIntegration.Db)
	suite.Require().NoError(err)
	err = MarkAsEmitted(context.Background(), suite.dbIntegration.Db, events)
	suite.Require().NoError(err)
	events, err = FindAllNewEvents(context.Background(), suite.dbIntegration.Db)
	suite.Require().NoError(err)
	suite.Require().Len(events, 0, "Should return 0 events")
}

// endregion tests
