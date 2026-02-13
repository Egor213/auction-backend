package config

import (
	"os"
	"time"

	errutils "auction-platform/pkg/errors"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

type (
	Config struct {
		App            `yaml:"app"`
		HTTP           `yaml:"http"`
		Log            `yaml:"log"`
		PG             `yaml:"postgres"`
		Kafka          `yaml:"kafka"`
		Redis          `yaml:"redis"`
		RateLimiter    `yaml:"rate_limiter"`
		Retry          `yaml:"retry"`
		CircuitBreaker `yaml:"circuit_breaker"`
	}

	App struct {
		Name    string `yaml:"name" env-required:"true"`
		Version string `yaml:"version" env-required:"true"`
	}

	HTTP struct {
		Address string `env-required:"true" env:"SERVER_ADDRESS"`
	}

	Log struct {
		Level string `yaml:"level" env:"LOG_LEVEL" env-default:"info"`
	}

	PG struct {
		URL         string `env-required:"true" env:"POSTGRES_CONN"`
		MaxPoolSize int    `env-required:"true" env:"MAX_POOL_SIZE" yaml:"max_pool_size"`
	}

	Kafka struct {
		Brokers         []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`
		BidPlacedTopic  string   `yaml:"bid_placed_topic"`
		BidResultTopic  string   `yaml:"bid_result_topic"`
		AuctionEndTopic string   `yaml:"auction_ended_topic"`
		GroupID         string   `yaml:"group_id"`
	}

	Redis struct {
		Addr     string        `env:"REDIS_ADDR" env-default:"localhost:6379"`
		Password string        `env:"REDIS_PASSWORD"`
		DB       int           `env:"REDIS_DB" env-default:"0"`
		CacheTTL time.Duration `yaml:"cache_ttl" env-default:"5m"`
	}

	RateLimiter struct {
		RPS   float64 `yaml:"rps" env:"RATE_RPS"`
		Burst int     `yaml:"burst" env:"RATE_BURST"`
	}

	Retry struct {
		MaxAttempts int           `yaml:"max_attempts"`
		InitialWait time.Duration `yaml:"initial_wait"`
		MaxWait     time.Duration `yaml:"max_wait"`
		Multiplier  float64       `yaml:"multiplier"`
	}

	CircuitBreaker struct {
		MaxRequests  uint32        `yaml:"max_requests"`
		Interval     time.Duration `yaml:"interval"`
		Timeout      time.Duration `yaml:"timeout"`
		MinRequests  uint32        `yaml:"min_requests"`
		FailureRatio float64       `yaml:"failure_ratio"`
	}
)

func New() (*Config, error) {
	cfg := &Config{}

	if err := godotenv.Load("infra/.env"); err != nil {
		log.WithError(err).Info(".env file not found, using system environment")
	}

	pathToConfig, ok := os.LookupEnv("APP_CONFIG_PATH")
	if !ok || pathToConfig == "" {
		log.WithField("env_var", "APP_CONFIG_PATH").
			Info("Config path is not set, using default")
		pathToConfig = "config/config.yaml"
	}

	if err := cleanenv.ReadConfig(pathToConfig, cfg); err != nil {
		return nil, errutils.WrapPathErr(err)
	}

	if err := cleanenv.UpdateEnv(cfg); err != nil {
		return nil, errutils.WrapPathErr(err)
	}

	return cfg, nil
}
