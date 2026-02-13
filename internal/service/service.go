package service

import (
	"auction-platform/internal/infrastruct/circuitbreaker"
	kafkaclient "auction-platform/internal/infrastruct/kafka"
	"auction-platform/internal/infrastruct/retry"
	"auction-platform/internal/metrics"
	"auction-platform/internal/repo"
	"context"
	"time"

	e "auction-platform/internal/entity"
	kd "auction-platform/internal/infrastruct/kafka/dto"
	sd "auction-platform/internal/service/dto"

	"github.com/redis/go-redis/v9"
)

type Auctions interface {
	CreateAuction(ctx context.Context, in sd.CreateAuctionInput) (e.Auction, error)
	GetAuction(ctx context.Context, auctionID string) (e.Auction, error)
	ListActive(ctx context.Context, page, pageSize int) ([]e.Auction, int64, error)
	InvalidateCache(ctx context.Context, auctionID string)
}

type Bids interface {
	PlaceBid(ctx context.Context, in sd.PlaceBidInput) (e.Bid, error)
	ProcessBidEvent(ctx context.Context, event kd.BidPlacedEvent) error
	GetBidsByAuction(ctx context.Context, auctionID string, limit int) ([]e.Bid, error)
	GetHighestBid(ctx context.Context, auctionID string) (e.Bid, error)
	CountByAuction(ctx context.Context, auctionID string) (int, error)
}

type Services struct {
	Auctions
	Bids
}

type ServicesDependencies struct {
	Repos       *repo.Repositories
	Redis       *redis.Client
	Breaker     *circuitbreaker.CircuitBreaker
	Retryer     *retry.Retryer
	Producer    *kafkaclient.Producer
	Metrics     *metrics.Metrics
	CacheTTL    time.Duration
	BidTopic    string
	ResultTopic string
}

func NewServices(deps ServicesDependencies) *Services {
	return &Services{
		Auctions: NewAuctionService(
			deps.Repos.Auctions, deps.Redis, deps.Breaker,
			deps.Retryer, deps.Metrics, deps.CacheTTL,
		),
		Bids: NewBidService(
			deps.Repos.Auctions, deps.Repos.Bids, deps.Producer,
			deps.Redis, deps.Breaker, deps.Retryer, deps.Metrics,
			deps.BidTopic, deps.ResultTopic,
		),
	}
}
