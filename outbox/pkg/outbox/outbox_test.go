//go:build integration

package outbox

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"

	"github.com/arngrimur/bilcool_monolith/outbox/pkg/outbox/testdata"
)

type outBoxTestSuite struct {
	suite.Suite

	// region variables
	outboxDB testdb.SuiteDbIntegration
	//endregion variables
}

// region setup
func (suite *outBoxTestSuite) SetupSuite() {
	suite.outboxDB = testdb.SetupDatabase(suite.T(), testdata.OutboxTestConnUrlTemplate, testdata.FS)
}
func (suite *outBoxTestSuite) TearDownSuite() {
	go suite.outboxDB.CancelFunc()
	_ = suite.outboxDB.Db.Close()
	testcontainers.CleanupContainer(suite.T(), suite.outboxDB.PostgresContainer)
}
func (suite *outBoxTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *outBoxTestSuite) AfterTest(suiteName, testName string) {
	_ = suite.outboxDB.Db.QueryRow("DROP PUBLICATION outbox")
}
func (suite *outBoxTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuiteoutBox(t *testing.T) {
	suite.Run(t, new(outBoxTestSuite))
}

// endregion setup
// region tests
func (suite *outBoxTestSuite) TestCreatePublication() {
	p := CreatePublication{
		publication: publication{
			Name:         "outbox",
			DatabaseName: "apa",
			Tables:       []string{"apa", "bepa"},
		},
	}
	u, err := url.Parse(testdata.OutboxTestConnUrlTemplate)
	suite.Require().NoError(err)
	err = NewOutbox(context.Background(), u, PgOutputPlugin, []Publication{p})
	suite.Require().NoError(err)
	row := suite.outboxDB.Db.QueryRow("select count(*) from pg_publication_tables")
	count := 0
	row.Scan(&count)
	suite.Require().Equal(1, count)
}

// endregion tests
