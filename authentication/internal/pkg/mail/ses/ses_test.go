//go:build integration

package ses

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/suite"

	localconfig "github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
)

// sesTestSuite We need a licences for AWS SES in LocalStack to run this test
// Keeping it as a reminder to do it later ( = never ).
type sesTestSuite struct {
	suite.Suite

	// region variables.
	awsConfig aws.Config

	//endregion variables
}

// region setup
func (suite *sesTestSuite) SetupSuite() {
	suite.T().Skip("ses test skipped, we have to setup a localstack instance of ses to run this test. Missing licesnce")
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("eu-north-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(localconfig.AwsKey(), localconfig.AwsSecret(), "")))
	suite.Require().NoError(err)
	suite.awsConfig = cfg
}
func (suite *sesTestSuite) TearDownSuite()                        {}
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
func (suite *sesTestSuite) TestSendMail() {
	err := NewSeSender(suite.awsConfig, "bilcool.branno@gmail.com").SendSecurityToken(
		suite.T().Context(),
		"arngrimurbjarnason@gmail.com",
		"123456",
		"",
	)
	suite.Require().NoError(err)
	suite.T().Log(
		"Check your inbox for a message from  with the subject 'Your BilCool security code'",
		"and the body 'Your security code is: 123456\n\nThis code is valid for 10 minutes.'",
		"and make sure to check your spam folder.",
	)
}

// endregion tests
