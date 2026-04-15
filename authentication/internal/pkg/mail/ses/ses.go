package ses

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsses "github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	"github.com/arngrimur/bilcool_monolith/authentication/internal/pkg/mail"
)

type SesSender struct {
	client    *awsses.Client
	fromEmail string
}

func NewSeSender(cfg aws.Config, fromEmail string) *SesSender {
	sesClient := awsses.NewFromConfig(cfg)
	return &SesSender{
		client:    sesClient,
		fromEmail: fromEmail,
	}
}

func (s *SesSender) SendSecurityToken(ctx context.Context, email string, token string, locale string) error {
	subject, _, body := mail.SecurityTokenContent(token, locale)
	_, err := s.client.SendEmail(ctx, &awsses.SendEmailInput{
		FromEmailAddress: aws.String(s.fromEmail),
		Destination: &types.Destination{
			ToAddresses: []string{email},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: aws.String(subject)},
				Body: &types.Body{
					Text: &types.Content{Data: aws.String(body)},
				},
			},
		},
	})
	return err
}
