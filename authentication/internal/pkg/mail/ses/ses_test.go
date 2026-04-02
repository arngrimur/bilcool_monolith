//go:build integration

package ses

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	test_aws "github.com/arngrimur/bilcool_monolith/testing/aws"
)

// sesTestSuite We need a licences for AWS SES in LocalStack to run this test
// Keeping it as a reminder to do it later ( = never ).
type sesTestSuite struct {
	suite.Suite

	// region variables.
	cloud *test_aws.AwsLocalCloud

	//endregion variables
}

// region setup
func (suite *sesTestSuite) SetupSuite() {
	//suite.cloud = test_aws.SetupLocalCloud(suite.T(), "ses")
}
func (suite *sesTestSuite) TearDownSuite() {
	//suite.cloud.TearDown(suite.T())
}
func (suite *sesTestSuite) BeforeTest(suiteName, testName string) {}
func (suite *sesTestSuite) AfterTest(suiteName, testName string)  {}
func (suite *sesTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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
func TestRunSuiteses(t *testing.T) {
	suite.Run(t, new(sesTestSuite))
}

// endregion setup
// region tests
//func (suite *sesTestSuite) TestSendMail() {
//	NewSeSender(suite.cloud.CreateConfig(suite.T()), "bilcool.branno@gmail.com").SendSecurityToken(
//		suite.T().Context(),
//		"arngrimurbjarnason@gmail.com",
//		"123456",
//	)
//	suite.T().Log(
//		"Check your inbox for a message from  with the subject 'Your BilCool security code'",
//		"and the body 'Your security code is: 123456\n\nThis code is valid for 10 minutes.'",
//		"and make sure to check your spam folder.",
//	)
//}

// endregion tests
