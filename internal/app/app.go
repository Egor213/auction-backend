package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auction-platform/internal/config"
	httpapi "auction-platform/internal/controller/http/v1"
	"auction-platform/internal/infrastruct/circuitbreaker"
	kafkaclient "auction-platform/internal/infrastruct/kafka"
	kd "auction-platform/internal/infrastruct/kafka/dto"
	"auction-platform/internal/infrastruct/retry"
	"auction-platform/internal/metrics"
	"auction-platform/internal/repo"
	"auction-platform/internal/service"
	"auction-platform/internal/worker"
	"auction-platform/pkg/httpserver"
	"auction-platform/pkg/logger"
	"auction-platform/pkg/postgres"
	"auction-platform/pkg/validator"

	errutils "auction-platform/pkg/errors"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	k "github.com/segmentio/kafka-go"
	log "github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

func Run() {
	// Config
	cfg, err := config.New()
	if err != nil {
		log.Fatal(errutils.WrapPathErr(err))
	}

	// Logger
	logger.SetupLogger(cfg.Log.Level)
	log.Info("Logger has been set up")

	// Migrations
	Migrate(cfg.PG.URL)

	// DB
	log.Info("Connecting to DB...")
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.MaxPoolSize))
	if err != nil {
		log.Fatal(errutils.WrapPathErr(err))
	}
	defer pg.Close()
	log.Info("Connected to DB")

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatal(errutils.WrapPathErr(err))
	}
	log.Info("Connected to Redis")

	// Metrics
	m := metrics.New()

	// Circuit Breaker
	cb := circuitbreaker.New(m)
	cb.Register("postgres", gobreaker.Settings{
		MaxRequests: cfg.CircuitBreaker.MaxRequests,
		Interval:    cfg.CircuitBreaker.Interval,
		Timeout:     cfg.CircuitBreaker.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.CircuitBreaker.MinRequests {
				return false
			}
			return float64(counts.TotalFailures)/float64(counts.Requests) >= cfg.CircuitBreaker.FailureRatio
		},
	})
	cb.Register("kafka_producer", gobreaker.Settings{
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	// Retryer
	retryer := retry.New(retry.Config{
		MaxAttempts: cfg.Retry.MaxAttempts,
		InitialWait: cfg.Retry.InitialWait,
		MaxWait:     cfg.Retry.MaxWait,
		Multiplier:  cfg.Retry.Multiplier,
	}, m)

	// Repos
	repositories := repo.NewRepositories(pg)

	// Kafka Producer
	kafkaTopics := []string{cfg.Kafka.BidPlacedTopic, cfg.Kafka.BidResultTopic, cfg.Kafka.AuctionEndTopic}
	producer := kafkaclient.NewProducer(cfg.Kafka.Brokers, kafkaTopics, cb, retryer, m)
	defer producer.Close()

	// Services
	services := service.NewServices(service.ServicesDependencies{
		Repos:       repositories,
		Redis:       rdb,
		Breaker:     cb,
		Retryer:     retryer,
		Producer:    producer,
		Metrics:     m,
		BidTopic:    cfg.Kafka.BidPlacedTopic,
		ResultTopic: cfg.Kafka.BidResultTopic,
	})

	// Kafka Consumer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bidConsumer := kafkaclient.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.BidPlacedTopic,
		cfg.Kafka.GroupID,
		func(ctx context.Context, msg k.Message) error {
			event, err := kafkaclient.ParseMessage[kd.BidPlacedEvent](msg)
			if err != nil {
				log.Errorf("Failed to parse bid event: %v", err)
				return err
			}
			return services.Bids.ProcessBidEvent(ctx, event)
		},
		m,
	)
	defer bidConsumer.Close()
	go bidConsumer.Start(ctx)

	// Worker auction expiry checker
	bidProcessor := worker.NewBidProcessor(
		services.Bids, repositories.Auctions, repositories.Bids,
		producer, m, cfg.Kafka.AuctionEndTopic,
	)
	go bidProcessor.StartExpiryChecker(ctx)

	// Echo handler
	log.Info("Initializing handlers and routes")
	handler := echo.New()
	handler.Validator = validator.NewCustomValidator()
	httpapi.ConfigureRouter(handler, services, m, pg.Pool, rdb, cfg.RateLimiter.RPS, cfg.RateLimiter.Burst)

	// HTTP server
	log.Info("Starting http server")
	log.Debugf("Server port: %s", cfg.HTTP.Address)
	httpServer := httpserver.New(handler, httpserver.Address(cfg.HTTP.Address))

	// Graceful shutdown
	log.Info("Configuring graceful shutdown")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		log.Error(errutils.WrapPathErr(err))
	}

	log.Info("Shutting down")
	cancel()
	err = httpServer.Shutdown()
	if err != nil {
		log.Error(errutils.WrapPathErr(err))
	}
}
