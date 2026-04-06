package brevo

import (
	"context"
	"fmt"

	brevolib "github.com/getbrevo/brevo-go/lib"
	"github.com/rs/zerolog/log"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/config"
)

type BrevoClient struct {
	client *brevolib.APIClient
}

func NewBrevoClient() *BrevoClient {
	cfg := brevolib.NewConfiguration()
	cfg.AddDefaultHeader("api-key", config.BrevoAPIKey())

	brc := brevolib.NewAPIClient(cfg)
	return &BrevoClient{
		client: brc,
	}
}

func (b BrevoClient) SendSecurityToken(ctx context.Context, toEmail string, token string) error {
	mail := brevolib.SendSmtpEmail{
		Sender: &brevolib.SendSmtpEmailSender{
			Name:  "Bilcool",
			Email: config.FromSenderEmail(),
		},
		To:          []brevolib.SendSmtpEmailTo{{Email: toEmail, Name: toEmail}},
		Subject:     "Your BilCool security code",
		HtmlContent: "Your security code is: <div><b>" + token + "</b><div>This code is valid for 10 minutes.",
		TextContent: "Your security code is: " + token + "\n\nThis code is valid for 10 minutes.",
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
