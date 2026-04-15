package brevo

import (
	"context"
	"fmt"

	brevolib "github.com/getbrevo/brevo-go/lib"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
)

type BrevoClient struct {
	client *brevolib.APIClient
}

func NewSender() *BrevoClient {
	cfg := brevolib.NewConfiguration()
	cfg.AddDefaultHeader("api-key", config.BrevoAPIKey())

	return &BrevoClient{client: brevolib.NewAPIClient(cfg)}
}

func (b BrevoClient) SendSecurityToken(ctx context.Context, toEmail string, token string, locale string) error {
	subject, htmlContent, textContent := mail.SecurityTokenContent(token, locale)
	mail := brevolib.SendSmtpEmail{
		Sender: &brevolib.SendSmtpEmailSender{
			Name:  "Bilcool",
			Email: config.FromSenderEmail(),
		},
		To:          []brevolib.SendSmtpEmailTo{{Email: toEmail, Name: toEmail}},
		Subject:     subject,
		HtmlContent: htmlContent,
		TextContent: textContent,
	}
	_, response, err := b.client.TransactionalEmailsApi.SendTransacEmail(ctx, mail)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("error sending email")
		return err
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	log.Ctx(ctx).Error().Any("response", response).Msgf("unexpected status code: %d", response.StatusCode)
	return fmt.Errorf("unexpected status code: %d", response.StatusCode)
}
