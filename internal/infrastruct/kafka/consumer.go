package kafka

import (
	"auction-platform/internal/metrics"
	"context"
	"encoding/json"
	"fmt"
	"time"

	k "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type MessageHandler func(ctx context.Context, msg k.Message) error

type Consumer struct {
	reader  *k.Reader
	handler MessageHandler
	metrics *metrics.Metrics
	topic   string
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	handler MessageHandler,
	m *metrics.Metrics,
) *Consumer {
	reader := k.NewReader(k.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: time.Second,
		StartOffset:    k.LastOffset,
	})
	return &Consumer{reader: reader, handler: handler, metrics: m, topic: topic}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Infof("Starting kafka consumer [%s]", c.topic)

	for {
		select {
		case <-ctx.Done():
			log.Infof("Stopping kafka consumer [%s]", c.topic)
			return
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Errorf("Kafka fetch error [%s]: %v", c.topic, err)
				time.Sleep(time.Second)
				continue
			}

			start := time.Now()

			if err := c.handler(ctx, msg); err != nil {
				c.metrics.KafkaMessagesConsumed.WithLabelValues(c.topic, "error").Inc()
				log.Errorf("Kafka handle error [%s] offset=%d: %v", c.topic, msg.Offset, err)
			} else {
				c.metrics.KafkaMessagesConsumed.WithLabelValues(c.topic, "success").Inc()
			}

			c.metrics.KafkaConsumeLatency.WithLabelValues(c.topic).Observe(time.Since(start).Seconds())

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Errorf("Kafka commit error [%s]: %v", c.topic, err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func ParseMessage[T any](msg k.Message) (T, error) {
	var result T
	if err := json.Unmarshal(msg.Value, &result); err != nil {
		return result, fmt.Errorf("unmarshal message: %w", err)
	}
	return result, nil
}
