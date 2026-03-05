//go:build integration

package outbox

import (
	"fmt"
	"strings"
	"testing"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"
	"github.com/stretchr/testify/suite"

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
	suite.outboxDB = testdb.SetupDatabase(suite.T(), "outbox", testdata.FS)
}
func (suite *outBoxTestSuite) TearDownSuite() {
	suite.outboxDB.TearDown(suite.T())
}
func (suite *outBoxTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *outBoxTestSuite) AfterTest(suiteName, testName string)  {}
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

// endregion tests
