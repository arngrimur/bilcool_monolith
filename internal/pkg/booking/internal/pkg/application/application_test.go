//go:build integration

package application

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type applicationTestSuite struct {
	suite.Suite
	// region variables

	//endregion variables
}

// region setup
func (suite *applicationTestSuite) SetupSuite()                           {}
func (suite *applicationTestSuite) TearDownSuite()                        {}
func (suite *applicationTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *applicationTestSuite) AfterTest(suiteName, testName string)  {}
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

}

// endregion tests
