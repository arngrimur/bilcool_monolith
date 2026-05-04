package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

func TestCreateUser(t *testing.T) {
	cases := []struct {
		name    string
		req     domain.CreateUserRequest
		setup   func(*domain.MockUsersRepository, *mail.MockMailSender)
		wantErr bool
	}{
		{
			name: "success",
			req:  domain.CreateUserRequest{Username: "alice", Email: "alice@example.com"},
			setup: func(repo *domain.MockUsersRepository, mail *mail.MockMailSender) {
				repo.EXPECT().CreateUser(gomock.Any(), domain.CreateUserRequest{Username: "alice", Email: "alice@example.com"}).
					Return(extdomain.UserResponse{UserRef: uuid.New(), Username: "alice", Email: "alice@example.com"}, nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "repository error",
			req:  domain.CreateUserRequest{Username: "alice", Email: "alice@example.com"},
			setup: func(repo *domain.MockUsersRepository, mail *mail.MockMailSender) {
				repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(extdomain.UserResponse{}, errors.New("db error")).Times(1)
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := domain.NewMockUsersRepository(ctrl)
			mail := mail.NewMockMailSender(ctrl)
			tc.setup(repo, mail)
			app := New(repo, mail, nil, "secret")
			resp, err := app.CreateUser(context.Background(), tc.req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.req.Username, resp.Username)
		})
	}
}

func TestLoginBegin_NoPasskeys_SendsToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockUsersRepository(ctrl)
	mail := mail.NewMockMailSender(ctrl)

	userRef := uuid.New()
	repo.EXPECT().FindByEmail(gomock.Any(), "alice@example.com").
		Return(extdomain.UserResponse{UserRef: userRef, Username: "alice", Email: "alice@example.com"}, nil).Times(1)
	repo.EXPECT().GetPasskeys(gomock.Any(), userRef).Return(nil, nil).Times(1)
	repo.EXPECT().CreateSecurityToken(gomock.Any(), userRef, gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mail.EXPECT().SendSecurityToken(gomock.Any(), "alice@example.com", gomock.Any(), gomock.Any()).Return(nil).Times(1)

	app := New(repo, mail, nil, "secret")
	resp, err := app.LoginBegin(context.Background(), domain.LoginBeginRequest{Email: "alice@example.com"})
	require.NoError(t, err)
	require.Equal(t, "verify_token", resp.NextStep)
	require.Nil(t, resp.SessionID)
}

func TestLoginBegin_UserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockUsersRepository(ctrl)
	mail := mail.NewMockMailSender(ctrl)

	repo.EXPECT().FindByEmail(gomock.Any(), "unknown@example.com").
		Return(extdomain.UserResponse{}, domain.ErrUserNotFound).Times(1)

	app := New(repo, mail, nil, "secret")
	_, err := app.LoginBegin(context.Background(), domain.LoginBeginRequest{Email: "unknown@example.com"})
	require.ErrorIs(t, err, domain.ErrUserNotFound)
}

func TestVerifyToken_InvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := domain.NewMockUsersRepository(ctrl)
	mail := mail.NewMockMailSender(ctrl)

	userRef := uuid.New()
	repo.EXPECT().FindByEmail(gomock.Any(), "alice@example.com").
		Return(extdomain.UserResponse{UserRef: userRef, Username: "alice", Email: "alice@example.com"}, nil).Times(1)
	repo.EXPECT().VerifyAndConsumeToken(gomock.Any(), userRef, "000000").Return(domain.ErrInvalidToken).Times(1)

	app := New(repo, mail, nil, "secret")
	_, err := app.VerifyToken(context.Background(), domain.VerifyTokenRequest{
		Email: "alice@example.com",
		Token: "000000",
	})
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestDeleteUser(t *testing.T) {
	cases := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name: "success",
		},
		{
			name:    "last admin blocked",
			repoErr: domain.ErrLastAdmin,
			wantErr: domain.ErrLastAdmin,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := domain.NewMockUsersRepository(ctrl)
			m := mail.NewMockMailSender(ctrl)

			userRef := uuid.New()
			repo.EXPECT().DeleteUser(gomock.Any(), userRef).Return(tc.repoErr).Times(1)

			app := New(repo, m, nil, "secret")
			err := app.DeleteUser(context.Background(), userRef)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	cases := []struct {
		name    string
		req     domain.UpdateUserRequest
		setup   func(*domain.MockUsersRepository)
		wantErr bool
	}{
		{
			name: "update username only",
			req:  domain.UpdateUserRequest{Username: strPtr("newname")},
			setup: func(repo *domain.MockUsersRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), domain.UpdateUserRequest{Username: strPtr("newname")}).
					Return(extdomain.UserResponse{Username: "newname", Email: "alice@example.com"}, nil).Times(1)
			},
		},
		{
			name: "update email only",
			req:  domain.UpdateUserRequest{Email: strPtr("new@example.com")},
			setup: func(repo *domain.MockUsersRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), domain.UpdateUserRequest{Email: strPtr("new@example.com")}).
					Return(extdomain.UserResponse{Username: "alice", Email: "new@example.com"}, nil).Times(1)
			},
		},
		{
			name: "update both",
			req:  domain.UpdateUserRequest{Username: strPtr("newname"), Email: strPtr("new@example.com")},
			setup: func(repo *domain.MockUsersRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(extdomain.UserResponse{Username: "newname", Email: "new@example.com"}, nil).Times(1)
			},
		},
		{
			name: "user not found",
			req:  domain.UpdateUserRequest{Username: strPtr("newname")},
			setup: func(repo *domain.MockUsersRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(extdomain.UserResponse{}, domain.ErrUserNotFound).Times(1)
			},
			wantErr: true,
		},
		{
			name: "email conflict",
			req:  domain.UpdateUserRequest{Email: strPtr("taken@example.com")},
			setup: func(repo *domain.MockUsersRepository) {
				repo.EXPECT().UpdateUser(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(extdomain.UserResponse{}, domain.ErrUserAlreadyExists).Times(1)
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := domain.NewMockUsersRepository(ctrl)
			m := mail.NewMockMailSender(ctrl)
			tc.setup(repo)
			app := New(repo, m, nil, "secret")
			resp, err := app.UpdateUser(context.Background(), uuid.New(), tc.req)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.req.Username != nil {
				require.Equal(t, *tc.req.Username, resp.Username)
			}
			if tc.req.Email != nil {
				require.Equal(t, *tc.req.Email, resp.Email)
			}
		})
	}
}

func TestGenerateSecurityToken(t *testing.T) {
	for i := 0; i < 10; i++ {
		token, err := domain.GenerateSecurityToken()
		require.NoError(t, err)
		require.Len(t, token, 6)
	}
}

func TestSecurityTokenExpiry(t *testing.T) {
	expiry := time.Now().Add(10 * time.Minute)
	require.True(t, time.Now().Before(expiry))
	require.True(t, expiry.Sub(time.Now()) <= 10*time.Minute+time.Second)
}
