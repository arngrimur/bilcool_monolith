package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTopicAndQueue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "unset uses the default", value: "", want: "default-name"},
		{name: "blank uses the default", value: "   ", want: "default-name"},
		{name: "set overrides the default", value: "bilcool-production-bookings", want: "bilcool-production-bookings"},
		{name: "surrounding space is trimmed", value: " bilcool_users ", want: "bilcool_users"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(TopicEnvVar, tt.value)
			t.Setenv(QueueEnvVar, tt.value)
			require.Equal(t, tt.want, Topic("default-name"))
			require.Equal(t, tt.want, Queue("default-name"))
		})
	}
}

func TestTopicAndQueueAreIndependent(t *testing.T) {
	t.Setenv(TopicEnvVar, "a-topic")
	t.Setenv(QueueEnvVar, "")
	require.Equal(t, "a-topic", Topic("default-topic"))
	require.Equal(t, "default-queue", Queue("default-queue"))
}
