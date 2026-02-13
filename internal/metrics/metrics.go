package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPActiveRequests  prometheus.Gauge

	AuctionsCreated    prometheus.Counter
	AuctionsFinished   prometheus.Counter
	BidsPlaced         prometheus.Counter
	BidsAccepted       prometheus.Counter
	BidsRejected       prometheus.Counter
	ActiveAuctions     prometheus.Gauge
	BidAmountHistogram prometheus.Histogram

	KafkaMessagesProduced *prometheus.CounterVec
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaProduceErrors    *prometheus.CounterVec
	KafkaConsumeLatency   *prometheus.HistogramVec

	CircuitBreakerState *prometheus.GaugeVec
	CircuitBreakerTrips *prometheus.CounterVec

	RateLimiterRejected prometheus.Counter

	RetryAttempts  *prometheus.HistogramVec
	RetryExhausted *prometheus.CounterVec

	DBQueryDuration *prometheus.HistogramVec
	DBErrors        *prometheus.CounterVec

	RedisCacheHits         prometheus.Counter
	RedisCacheMisses       prometheus.Counter
	RedisOperationDuration *prometheus.HistogramVec
}

func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_http_requests_total",
			Help: "Total HTTP requests",
		}, []string{"method", "path", "status"}),

		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "auction_http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"method", "path"}),

		HTTPActiveRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "auction_http_active_requests",
			Help: "Active HTTP requests",
		}),

		AuctionsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_auctions_created_total",
		}),
		AuctionsFinished: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_auctions_finished_total",
		}),
		BidsPlaced: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_bids_placed_total",
		}),
		BidsAccepted: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_bids_accepted_total",
		}),
		BidsRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_bids_rejected_total",
		}),
		ActiveAuctions: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "auction_active_auctions",
		}),
		BidAmountHistogram: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "auction_bid_amount",
			Buckets: []float64{1, 5, 10, 50, 100, 500, 1000, 5000, 10000},
		}),

		KafkaMessagesProduced: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_kafka_produced_total",
		}, []string{"topic"}),
		KafkaMessagesConsumed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_kafka_consumed_total",
		}, []string{"topic", "status"}),
		KafkaProduceErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_kafka_produce_errors_total",
		}, []string{"topic"}),
		KafkaConsumeLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "auction_kafka_consume_latency_seconds",
			Buckets: []float64{.001, .005, .01, .05, .1, .5, 1, 5},
		}, []string{"topic"}),

		CircuitBreakerState: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "auction_cb_state",
		}, []string{"name"}),
		CircuitBreakerTrips: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_cb_trips_total",
		}, []string{"name"}),

		RateLimiterRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_rate_limiter_rejected_total",
		}),

		RetryAttempts: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "auction_retry_attempts",
			Buckets: []float64{1, 2, 3, 4, 5},
		}, []string{"operation"}),
		RetryExhausted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_retry_exhausted_total",
		}, []string{"operation"}),

		DBQueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "auction_db_query_duration_seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}, []string{"query"}),
		DBErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "auction_db_errors_total",
		}, []string{"query"}),

		RedisCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_redis_hits_total",
		}),
		RedisCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "auction_redis_misses_total",
		}),
		RedisOperationDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "auction_redis_op_duration_seconds",
			Buckets: []float64{.0005, .001, .005, .01, .025, .05, .1},
		}, []string{"operation"}),
	}
}
