//go:build integration

package outbox

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"

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
	zerolog.SetGlobalLevel(zerolog.NoLevel)
}
func (suite *outBoxTestSuite) TearDownSuite() {

}
func (suite *outBoxTestSuite) BeforeTest(suiteName, testName string) {
	outboxOnce = sync.Once{}
	outboxInstance = nil
	outboxErr = nil
	suite.outboxDB = testdb.SetupDatabase(suite.T(), testdata.OutboxTestConnUrlTemplate, testdata.FS)
}
func (suite *outBoxTestSuite) AfterTest(suiteName, testName string) {
	suite.outboxDB.TearDown(suite.T())
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
			PublicationName: "outbox_test_pub",
			DatabaseName:    "outbox",
			Tables:          []string{"apa", "bepa"},
		},
	}
	outbox, err := NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	_, err = outbox.StartReplication()
	suite.Require().NoError(err)
	//defer close(stopChannel)
	row := suite.outboxDB.Db.QueryRow("select count(*) from pg_publication_tables")
	count := 0
	row.Scan(&count)
	suite.Require().Equal(2, count)
}

func (suite *outBoxTestSuite) TestCreatePublicationAfterStop() {
	p := CreatePublication{
		publication: publication{
			PublicationName:  "outbox_test_pub",
			DatabaseName:     "outbox",
			Tables:           []string{"apa", "bepa"},
			RegisterdActions: &Actions{},
		},
	}

	_, err := newOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	_, err = NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
}

func (suite *outBoxTestSuite) TestProcessData() {
	mockCtrl := gomock.NewController(suite.T())
	actions := NewActions()
	action := NewMockAction(mockCtrl)
	action.EXPECT().Execute(gomock.Any()).Return().Times(1)
	actions.Add(ActionInsert, action)
	p := CreatePublication{
		publication: publication{
			PublicationName:  "outbox_test_pub",
			DatabaseName:     "outbox",
			Tables:           []string{"apa"},
			RegisterdActions: actions,
		},
	}
	o, err := NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	cancelF, err := o.StartReplication()
	suite.Require().NoError(err)
	defer func() {
		time.Sleep(3 * time.Second)
		close(cancelF)
	}()
	_, err = suite.outboxDB.Db.Exec("INSERT INTO apa VALUES (100, 'apa')")
	suite.Require().NoError(err)
	_, err = suite.outboxDB.Db.Exec("INSERT INTO bepa VALUES (200, 'bepa')")
	suite.Require().NoError(err)

}

// endregion tests
