// Package logsender is a development-only mail.MailSender that writes the security token to the log instead of
// sending a mail. It is selected with MAIL_SENDER=log and must never be used in production.
package logsender

import (
	"context"

	"github.com/rs/zerolog/log"
)

type LogSender struct{}

func NewSender() *LogSender {
	return &LogSender{}
}

func (LogSender) SendSecurityToken(ctx context.Context, toEmail string, token string, locale string) error {
	log.Ctx(ctx).Warn().Str("to", toEmail).Str("token", token).Str("locale", locale).
		Msg("DEV ONLY (MAIL_SENDER=log): security token not mailed")
	return nil
}
