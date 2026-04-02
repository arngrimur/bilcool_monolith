package ses

import "context"

//go:generate mockgen -source=mail_sender.go -destination=mail_sender_mock.go -package=ses
type MailSender interface {
	SendSecurityToken(ctx context.Context, email, token string) error
}
