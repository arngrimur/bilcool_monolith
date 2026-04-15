package mail

import "context"

//go:generate mockgen -source=mail_sender.go -destination=mail_sender_mock.go -package=mail
type MailSender interface {
	SendSecurityToken(ctx context.Context, toEmail string, token string, locale string) error
}
