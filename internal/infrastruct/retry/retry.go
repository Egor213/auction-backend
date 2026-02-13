package retry

import (
	"auction-platform/internal/metrics"
	errutils "auction-platform/pkg/errors"
	"context"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Multiplier  float64
}

type Retryer struct {
	cfg     Config
	metrics *metrics.Metrics
}

func New(cfg Config, m *metrics.Metrics) *Retryer {
	return &Retryer{cfg: cfg, metrics: m}
}

func (r *Retryer) Do(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.cfg.MaxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			r.metrics.RetryAttempts.WithLabelValues(operation).Observe(float64(attempt))
			return nil
		}

		if attempt == r.cfg.MaxAttempts {
			break
		}

		wait := r.backoff(attempt)
		log.Warnf("Retry [%s] attempt %d/%d, wait %s, err: %v", operation, attempt, r.cfg.MaxAttempts, wait, lastErr)

		select {
		case <-ctx.Done():
			return fmt.Errorf("retry canceled: %w", ctx.Err())
		case <-time.After(wait):
		}
	}

	r.metrics.RetryExhausted.WithLabelValues(operation).Inc()
	log.Error(errutils.WrapPathErr(fmt.Errorf("retry exhausted [%s]: %w", operation, lastErr)))
	return fmt.Errorf("after %d attempts: %w", r.cfg.MaxAttempts, lastErr)
}

func (r *Retryer) backoff(attempt int) time.Duration {
	b := min(float64(r.cfg.InitialWait)*math.Pow(r.cfg.Multiplier, float64(attempt-1)), float64(r.cfg.MaxWait))
	return time.Duration(b)
}
