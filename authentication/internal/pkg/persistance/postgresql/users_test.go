//go:build integration

package postgresql

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	_ "github.com/lib/pq"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/migrations"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	coutbox "github.com/arngrimur/bilcool_monolith/message_broker/pkg/postgres"
	"github.com/arngrimur/bilcool_monolith/testing/testdb"
)

type usersTestSuite struct {
	suite.Suite
	testdb.SuiteDbIntegration
	userRef  uuid.UUID
	username string
	email    string
}

func (suite *usersTestSuite) SetupSuite() {
	suite.SuiteDbIntegration = testdb.SetupDatabase(suite.T(), migrations.FS, "authentication_test")
	err := coutbox.CreateTable(suite.Db)
	suite.Require().NoError(err)
	suite.username = "testuser"
	suite.email = "test@example.com"
}

func (suite *usersTestSuite) TearDownSuite() {
	suite.SuiteDbIntegration.TearDown(suite.T())
}

func (suite *usersTestSuite) BeforeTest(suiteName, testName string) {
	repo := NewUsersRepository(suite.Db)
	resp, err := repo.CreateUser(context.Background(), domain.CreateUserRequest{
		Username: suite.username,
		Email:    suite.email,
	})
	suite.Require().NoError(err)
	suite.userRef = resp.UserRef
}

func (suite *usersTestSuite) AfterTest(suiteName, testName string) {
	_, err := suite.Db.Exec("TRUNCATE TABLE webauthn_sessions, security_tokens, passkeys, users CASCADE")
	suite.Require().NoError(err)
}

func (suite *usersTestSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
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

func TestRunSuiteUsers(t *testing.T) {
	suite.Run(t, new(usersTestSuite))
}

func (suite *usersTestSuite) TestCreateUserAndFindByEmail() {
	repo := NewUsersRepository(suite.Db)
	user, err := repo.FindByEmail(context.Background(), suite.email)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.username, user.Username)
	suite.Require().Equal(suite.email, user.Email)
}

func (suite *usersTestSuite) TestFindByRef() {
	repo := NewUsersRepository(suite.Db)
	user, err := repo.FindByRef(context.Background(), suite.userRef)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.userRef, user.UserRef)
}

func (suite *usersTestSuite) TestDeleteUser() {
	repo := NewUsersRepository(suite.Db)
	err := repo.DeleteUser(context.Background(), suite.userRef)
	suite.Require().NoError(err)
	_, err = repo.FindByEmail(context.Background(), suite.email)
	suite.Require().Error(err)
}

func (suite *usersTestSuite) TestDeleteLastAdminIsRejected() {
	repo := NewUsersRepository(suite.Db)

	// Create a single admin — the only admin in the system
	admin, err := repo.CreateUser(context.Background(), domain.CreateUserRequest{
		Username: "soleadmin",
		Email:    "soleadmin@example.com",
	})
	suite.Require().NoError(err)
	err = repo.ChangeUserRole(context.Background(), admin.UserRef, "admin")
	suite.Require().NoError(err)

	// Deleting this admin should fail because they are the last admin
	err = repo.DeleteUser(context.Background(), admin.UserRef)
	suite.Require().ErrorIs(err, domain.ErrLastAdmin)
}

func (suite *usersTestSuite) TestDeleteAdminWhenAnotherAdminExists() {
	repo := NewUsersRepository(suite.Db)

	// Create two admin users
	first, err := repo.CreateUser(context.Background(), domain.CreateUserRequest{
		Username: "firstadmin",
		Email:    "firstadmin@example.com",
	})
	suite.Require().NoError(err)
	err = repo.ChangeUserRole(context.Background(), first.UserRef, "admin")
	suite.Require().NoError(err)

	second, err := repo.CreateUser(context.Background(), domain.CreateUserRequest{
		Username: "secondadmin",
		Email:    "secondadmin@example.com",
	})
	suite.Require().NoError(err)
	err = repo.ChangeUserRole(context.Background(), second.UserRef, "admin")
	suite.Require().NoError(err)

	// Deleting the first admin should succeed since a second admin exists
	err = repo.DeleteUser(context.Background(), first.UserRef)
	suite.Require().NoError(err)

	_, err = repo.FindByEmail(context.Background(), "firstadmin@example.com")
	suite.Require().Error(err)
}

func (suite *usersTestSuite) TestSecurityToken() {
	repo := NewUsersRepository(suite.Db)
	token := "123456"
	expiresAt := time.Now().Add(10 * time.Minute)

	err := repo.CreateSecurityToken(context.Background(), suite.userRef, token, expiresAt)
	suite.Require().NoError(err)

	err = repo.VerifyAndConsumeToken(context.Background(), suite.userRef, token)
	suite.Require().NoError(err)

	err = repo.VerifyAndConsumeToken(context.Background(), suite.userRef, token)
	suite.Require().ErrorIs(err, domain.ErrInvalidToken)
}

func (suite *usersTestSuite) TestExpiredToken() {
	repo := NewUsersRepository(suite.Db)
	token := "999999"
	expiresAt := time.Now().Add(-1 * time.Minute)

	err := repo.CreateSecurityToken(context.Background(), suite.userRef, token, expiresAt)
	suite.Require().NoError(err)

	err = repo.VerifyAndConsumeToken(context.Background(), suite.userRef, token)
	suite.Require().ErrorIs(err, domain.ErrInvalidToken)
}

func (suite *usersTestSuite) TestStoreAndGetPasskey() {
	repo := NewUsersRepository(suite.Db)
	passkey := domain.Passkey{
		CredentialID: []byte("test-credential-id"),
		Data:         []byte(`{"id":"dGVzdA==","public_key":"dGVzdA=="}`),
	}

	err := repo.StorePasskey(context.Background(), suite.userRef, passkey)
	suite.Require().NoError(err)

	passkeys, err := repo.GetPasskeys(context.Background(), suite.userRef)
	suite.Require().NoError(err)
	suite.Require().Len(passkeys, 1)
	suite.Require().Equal(passkey.CredentialID, passkeys[0].CredentialID)
}

func (suite *usersTestSuite) TestWebAuthnSession() {
	repo := NewUsersRepository(suite.Db)
	sessionID := uuid.New()
	session := domain.WebAuthnSession{
		SessionID:   sessionID,
		UserRef:     suite.userRef,
		SessionType: "registration",
		Data:        []byte(`{"challenge":"dGVzdA=="}`),
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}

	err := repo.StoreWebAuthnSession(context.Background(), session)
	suite.Require().NoError(err)

	loaded, err := repo.GetWebAuthnSession(context.Background(), sessionID)
	suite.Require().NoError(err)
	suite.Require().Equal(sessionID, loaded.SessionID)
	suite.Require().Equal("registration", loaded.SessionType)

	err = repo.DeleteWebAuthnSession(context.Background(), sessionID)
	suite.Require().NoError(err)

	_, err = repo.GetWebAuthnSession(context.Background(), sessionID)
	suite.Require().ErrorIs(err, domain.ErrSessionNotFound)
}
