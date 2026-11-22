package provider

import (
	"os"
	"strings"
)

const (
	// TopicEnvVar names the environment variable that overrides the topic a service publishes to.
	TopicEnvVar = "MESSAGING_TOPIC"
	// QueueEnvVar names the environment variable that overrides the queue a service consumes from.
	QueueEnvVar = "MESSAGING_QUEUE"
)

// Topic returns the topic this process publishes to: MESSAGING_TOPIC, or defaultName when it is unset or blank.
// Each service publishes to exactly one topic, so one variable per process is enough.
func Topic(defaultName string) string {
	return fromEnv(TopicEnvVar, defaultName)
}

// Queue returns the queue this process consumes from: MESSAGING_QUEUE, or defaultName when it is unset or blank.
// Each service consumes from exactly one queue, so one variable per process is enough.
func Queue(defaultName string) string {
	return fromEnv(QueueEnvVar, defaultName)
}

func fromEnv(name, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return defaultValue
}
