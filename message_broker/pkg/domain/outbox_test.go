//go:build integration

package domain

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"

	"github.com/arngrimur/bilcool_monolith/testing/testdb"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres/testdata"
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
	suite.outboxDB = testdb.SetupDatabase(suite.T(), testdata.FS, testName)
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
		PublicationBase: PublicationBase{
			PublicationName: "outbox_test_pub",
			DatabaseName:    "outbox",
			Tables:          []string{"apa", "bepa"},
		},
	}
	outbox, err := NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	stopChannel, err := outbox.StartReplication(context.Background())
	suite.Require().NoError(err)
	defer close(stopChannel)
	row := suite.outboxDB.Db.QueryRow("select count(*) from pg_publication_tables")
	count := 0
	row.Scan(&count)
	suite.Require().Equal(2, count)
}

func (suite *outBoxTestSuite) TestCreatePublicationAfterStop() {
	p := CreatePublication{
		PublicationBase: PublicationBase{
			PublicationName:   "outbox_test_pub",
			DatabaseName:      "outbox",
			Tables:            []string{"apa", "bepa"},
			RegisteredActions: &Actions{},
		},
	}

	_, err := newOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	_, err = NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)

}

func (suite *outBoxTestSuite) TestProcessData() {
	mockCtrl := gomock.NewController(suite.T())

	postgres.CreateTable(suite.outboxDB.Db)

	actions := NewActions()
	insertAction := NewMockAction(mockCtrl)
	insertAction.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(2)
	actions.RegisterAction(ActionInsert, insertAction)
	commitAction := NewMockAction(mockCtrl)
	commitAction.EXPECT().Execute(gomock.Any(), gomock.Any()).Times(3)
	actions.RegisterAction(ActionCommit, commitAction)
	p := CreatePublication{
		PublicationBase: PublicationBase{
			PublicationName:   "outbox_test_pub",
			DatabaseName:      "outbox",
			Tables:            []string{"apa", "outbox"},
			RegisteredActions: actions,
		},
	}
	o, err := NewOutbox(context.Background(), suite.outboxDB.ConnString, PgOutputPlugin, p)
	suite.Require().NoError(err)
	cancelF, err := o.StartReplication(context.Background())
	suite.Require().NoError(err)
	defer func() {
		time.Sleep(3 * time.Second)
		close(cancelF)
	}()
	_, err = suite.outboxDB.Db.Exec("INSERT INTO apa VALUES (100, 'apa')")
	suite.Require().NoError(err)
	_, err = suite.outboxDB.Db.Exec("INSERT INTO bepa VALUES (200, 'bepa')") // shall not create any WAL execution
	suite.Require().NoError(err)
	tx, err := suite.outboxDB.Db.BeginTx(context.Background(), nil)
	suite.Require().NoError(err)
	_, err = tx.Exec(`INSERT INTO outbox(id,event_id,type,correlation_id,producer,payload) 
VALUES (2, $1,'commit',$2,'test',$3 )`, uuid.New(), uuid.New(), []byte(`{"test":"data"}`))
	suite.Require().NoError(err)
	err = tx.Commit()
	suite.Require().NoError(err)
}

// endregion tests
