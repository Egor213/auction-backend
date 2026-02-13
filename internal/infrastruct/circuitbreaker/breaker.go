package circuitbreaker

import (
	"auction-platform/internal/metrics"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

type CircuitBreaker struct {
	breakers map[string]*gobreaker.CircuitBreaker
	metrics  *metrics.Metrics
}

func New(m *metrics.Metrics) *CircuitBreaker {
	return &CircuitBreaker{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		metrics:  m,
	}
}

func (cb *CircuitBreaker) Register(name string, settings gobreaker.Settings) {
	settings.Name = name
	settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
		log.Warnf("Circuit breaker [%s]: %s -> %s", name, from.String(), to.String())

		stateVal := float64(0)
		switch to {
		case gobreaker.StateHalfOpen:
			stateVal = 1
		case gobreaker.StateOpen:
			stateVal = 2
			cb.metrics.CircuitBreakerTrips.WithLabelValues(name).Inc()
		}
		cb.metrics.CircuitBreakerState.WithLabelValues(name).Set(stateVal)
	}

	cb.breakers[name] = gobreaker.NewCircuitBreaker(settings)
	cb.metrics.CircuitBreakerState.WithLabelValues(name).Set(0)
}

func (cb *CircuitBreaker) Execute(name string, fn func() (any, error)) (any, error) {
	breaker, ok := cb.breakers[name]
	if !ok {
		return nil, fmt.Errorf("circuit breaker %q not registered", name)
	}
	return breaker.Execute(fn)
}
