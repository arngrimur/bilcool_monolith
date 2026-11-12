// Package provider selects the concrete messaging implementation from the environment.
package provider

import (
	"context"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/messaging"
	"github.com/arngrimur/bilcool_monolith/message_broker/pkg/outbox/sns"
)

const (
	// EnvVar names the environment variable that selects the implementation.
	EnvVar = "MESSAGING_PROVIDER"

	// SnsSqs publishes to AWS SNS topics and consumes from AWS SQS queues. It is the default.
	SnsSqs = "sns-sqs"
)

// Name returns the selected implementation: the value of MESSAGING_PROVIDER, or SnsSqs when it is unset or empty.
func Name() (string, error) {
	switch name := os.Getenv(EnvVar); name {
	case "", SnsSqs:
		return SnsSqs, nil
	default:
		return "", fmt.Errorf("%s=%q is not supported (supported: %q)", EnvVar, name, SnsSqs)
	}
}

// NewPublisher creates the Publisher of the selected implementation. Each implementation reads its own
// connection settings from the environment (for sns-sqs: the AWS SDK default configuration).
func NewPublisher(ctx context.Context) (messaging.Publisher, error) {
	name, err := Name()
	if err != nil {
		return nil, err
	}
	switch name {
	case SnsSqs:
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}
		return sns.NewPublisher(ctx, cfg)
	default:
		return nil, fmt.Errorf("no publisher for %s=%q", EnvVar, name)
	}
}
