package sns

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	awssns "github.com/aws/aws-sdk-go-v2/service/sns"
)

type TopicCache struct {
	mu        sync.RWMutex
	topics    map[string]string
	snsClient *awssns.Client
}

func CreateCache(ctx context.Context, snsClient *awssns.Client) (*TopicCache, error) {
	c := &TopicCache{
		snsClient: snsClient,
		topics:    make(map[string]string),
	}
	if err := c.populate(ctx); err != nil {
		return nil, err
	}
	go c.refreshWorker(ctx)
	return c, nil
}

func (c *TopicCache) refreshWorker(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = c.populate(ctx)
		}
	}
}

func (c *TopicCache) populate(ctx context.Context) error {
	topics := make(map[string]string)
	input := &awssns.ListTopicsInput{}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		result, err := c.snsClient.ListTopics(ctx, input)
		if err != nil {
			return err
		}
		for _, t := range result.Topics {
			arn := *t.TopicArn
			parts := strings.Split(arn, ":")
			topics[parts[len(parts)-1]] = arn
		}
		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}
	c.mu.Lock()
	c.topics = topics
	c.mu.Unlock()
	return nil
}

func (c *TopicCache) GetTopicArn(ctx context.Context, topic string) (*string, error) {
	c.mu.RLock()
	arn, ok := c.topics[topic]
	c.mu.RUnlock()
	if ok {
		return &arn, nil
	}
	if err := c.populate(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	arn, ok = c.topics[topic]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("topic %s not found", topic)
	}
	return &arn, nil
}