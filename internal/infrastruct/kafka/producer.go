package kafka

import (
	"auction-platform/internal/infrastruct/circuitbreaker"
	"auction-platform/internal/infrastruct/retry"
	"auction-platform/internal/metrics"
	errutils "auction-platform/pkg/errors"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
)

type Producer struct {
	writers map[string]*kafka.Writer
	breaker *circuitbreaker.CircuitBreaker
	retryer *retry.Retryer
	metrics *metrics.Metrics
}

func NewProducer(
	brokers []string,
	topics []string,
	breaker *circuitbreaker.CircuitBreaker,
	retryer *retry.Retryer,
	m *metrics.Metrics,
) *Producer {
	writers := make(map[string]*kafka.Writer)
	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			BatchSize:    100,
			Async:        false,
			RequiredAcks: kafka.RequireOne,
		}
	}
	return &Producer{writers: writers, breaker: breaker, retryer: retryer, metrics: m}
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errutils.WrapPathErr(err)
	}

	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("unknown topic: %s", topic)
	}

	_, cbErr := p.breaker.Execute("kafka_producer", func() (any, error) {
		retryErr := p.retryer.Do(ctx, "kafka_produce_"+topic, func() error {
			return writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(key),
				Value: data,
			})
		})
		return nil, retryErr
	})

	if cbErr != nil {
		p.metrics.KafkaProduceErrors.WithLabelValues(topic).Inc()
		log.Error(errutils.WrapPathErr(cbErr))
		return errutils.WrapPathErr(cbErr)
	}

	p.metrics.KafkaMessagesProduced.WithLabelValues(topic).Inc()
	return nil
}

func (p *Producer) Close() {
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			log.Errorf("Failed to close kafka writer [%s]: %v", topic, err)
		}
	}
}
