package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "unset defaults to sns-sqs", value: "", want: SnsSqs},
		{name: "sns-sqs", value: "sns-sqs", want: SnsSqs},
		{name: "unknown is rejected", value: "kafka", wantErr: true},
		{name: "case sensitive", value: "SNS-SQS", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvVar, tt.value)
			got, err := Name()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), EnvVar)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNewPublisherRejectsUnknownProvider(t *testing.T) {
	t.Setenv(EnvVar, "kafka")
	publisher, err := NewPublisher(t.Context())
	require.Error(t, err)
	require.Nil(t, publisher)
}

func TestNewSubscriberRejectsUnknownProvider(t *testing.T) {
	t.Setenv(EnvVar, "kafka")
	subscriber, err := NewSubscriber(t.Context(), "a-queue")
	require.Error(t, err)
	require.Nil(t, subscriber)
}
