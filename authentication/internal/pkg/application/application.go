package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application/commands"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/application/queries"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/domain"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
	extdomain "github.com/arngrimur/bilcool_monolith/authentication/pkg/domain"
)

type (
	App interface {
		Commands
		Queries
	}

	Commands interface {
		CreateUser(ctx context.Context, req domain.CreateUserRequest) (extdomain.UserResponse, error)
		DeleteUser(ctx context.Context, userRef uuid.UUID) error
		RestoreUser(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error)
		UpdateUser(ctx context.Context, userRef uuid.UUID, req domain.UpdateUserRequest) (extdomain.UserResponse, error)
		ChangeUserRole(ctx context.Context, callerRef, targetRef uuid.UUID, newRole string) error
		LoginBegin(ctx context.Context, req domain.LoginBeginRequest) (domain.LoginBeginResponse, error)
		VerifyToken(ctx context.Context, req domain.VerifyTokenRequest) (domain.VerifyTokenResponse, error)
		LoginComplete(ctx context.Context, req domain.LoginCompleteRequest) (domain.LoginCompleteResponse, error)
		ResetLogin(ctx context.Context, req domain.ResetLoginRequest) error
	}

	Queries interface {
		ListUsers(ctx context.Context) ([]extdomain.UserResponse, error)
		ListDeletedUsers(ctx context.Context) ([]extdomain.DeletedUserResponse, error)
		GetUserByRef(ctx context.Context, userRef uuid.UUID) (extdomain.UserResponse, error)
	}
)

type (
	Application struct {
		appCommands
		appQueries
	}
	appCommands struct {
		commands.CreateUserHandler
		commands.DeleteUserHandler
		commands.RestoreUserHandler
		commands.UpdateUserHandler
		commands.ChangeUserRoleHandler
		commands.LoginBeginHandler
		commands.VerifyTokenHandler
		commands.LoginCompleteHandler
		commands.ResetLoginHandler
	}
	appQueries struct {
		queries.ListUsersHandler
		queries.ListDeletedUsersHandler
		queries.GetUserHandler
	}
)

var _ App = (*Application)(nil)

func New(
	usersRepo domain.UsersRepository,
	mailSender mail.MailSender,
	webAuthn domain.WebAuthnProvider,
	jwtSecret string,
) *Application {
	users := domain.NewUsers(usersRepo, mailSender)
	return &Application{
		appCommands: appCommands{
			CreateUserHandler:     commands.NewCreateUserHandler(users),
			DeleteUserHandler:     commands.NewDeleteUserHandler(users),
			RestoreUserHandler:    commands.NewRestoreUserHandler(users),
			UpdateUserHandler:     commands.NewUpdateUserHandler(users),
			ChangeUserRoleHandler: commands.NewChangeUserRoleHandler(users),
			LoginBeginHandler:     commands.NewLoginBeginHandler(users, webAuthn),
			VerifyTokenHandler:    commands.NewVerifyTokenHandler(users, webAuthn),
			LoginCompleteHandler:  commands.NewLoginCompleteHandler(users, webAuthn, jwtSecret),
			ResetLoginHandler:     commands.NewResetLoginHandler(users),
		},
		appQueries: appQueries{
			ListUsersHandler:        queries.NewListUsersHandler(users),
			ListDeletedUsersHandler: queries.NewListDeletedUsersHandler(users),
			GetUserHandler:          queries.NewGetUserHandler(users),
		},
	}
}
