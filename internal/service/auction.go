package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	e "auction-platform/internal/entity"
	"auction-platform/internal/infrastruct/circuitbreaker"
	"auction-platform/internal/infrastruct/retry"
	"auction-platform/internal/metrics"
	"auction-platform/internal/repo"
	re "auction-platform/internal/repo/errors"
	sd "auction-platform/internal/service/dto"
	se "auction-platform/internal/service/errors"
	smap "auction-platform/internal/service/mappers"
	errutils "auction-platform/pkg/errors"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
)

type AuctionService struct {
	auctionRepo repo.Auctions
	redis       *redis.Client
	breaker     *circuitbreaker.CircuitBreaker
	retryer     *retry.Retryer
	metrics     *metrics.Metrics
	cacheTTL    time.Duration
}

func NewAuctionService(
	aRepo repo.Auctions,
	rdb *redis.Client,
	breaker *circuitbreaker.CircuitBreaker,
	retryer *retry.Retryer,
	m *metrics.Metrics,
	cacheTTL time.Duration,
) *AuctionService {
	return &AuctionService{
		auctionRepo: aRepo,
		redis:       rdb,
		breaker:     breaker,
		retryer:     retryer,
		metrics:     m,
		cacheTTL:    cacheTTL,
	}
}

func (s *AuctionService) CreateAuction(ctx context.Context, in sd.CreateAuctionInput) (e.Auction, error) {
	repoIn := smap.ToCreateAuctionRepoInput(in)

	result, cbErr := s.breaker.Execute("postgres", func() (any, error) {
		var auction e.Auction
		err := s.retryer.Do(ctx, "create_auction", func() error {
			var e error
			auction, e = s.auctionRepo.Create(ctx, repoIn)
			return e
		})
		return auction, err
	})
	if cbErr != nil {
		log.Error(errutils.WrapPathErr(cbErr))
		if errors.Is(cbErr, re.ErrAlreadyExists) {
			return e.Auction{}, se.ErrAuctionAlreadyExists
		}
		return e.Auction{}, se.ErrCannotCreateAuction
	}

	auction := result.(e.Auction)
	s.cacheAuction(ctx, auction)
	s.metrics.AuctionsCreated.Inc()
	s.metrics.ActiveAuctions.Inc()

	return auction, nil
}

func (s *AuctionService) GetAuction(ctx context.Context, auctionID string) (e.Auction, error) {
	if cached := s.getCachedAuction(ctx, auctionID); cached != nil {
		s.metrics.RedisCacheHits.Inc()
		return *cached, nil
	}
	s.metrics.RedisCacheMisses.Inc()

	result, cbErr := s.breaker.Execute("postgres", func() (any, error) {
		var auction e.Auction
		err := s.retryer.Do(ctx, "get_auction", func() error {
			var e error
			auction, e = s.auctionRepo.GetByID(ctx, auctionID)
			return e
		})
		return auction, err
	})
	if cbErr != nil {
		log.Error(errutils.WrapPathErr(cbErr))
		return e.Auction{}, se.HandleRepoNotFound(cbErr, se.ErrNotFoundAuction, se.ErrCannotGetAuction)
	}

	auction := result.(e.Auction)
	s.cacheAuction(ctx, auction)
	return auction, nil
}

func (s *AuctionService) ListActive(ctx context.Context, page, pageSize int) ([]e.Auction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	result, cbErr := s.breaker.Execute("postgres", func() (any, error) {
		type la struct {
			auctions []e.Auction
			total    int64
		}
		var r la
		err := s.retryer.Do(ctx, "list_auctions", func() error {
			var e error
			r.auctions, r.total, e = s.auctionRepo.ListActive(ctx, pageSize, offset)
			return e
		})
		return r, err
	})
	if cbErr != nil {
		log.Error(errutils.WrapPathErr(cbErr))
		return nil, 0, se.ErrCannotListAuctions
	}

	type la struct {
		auctions []e.Auction
		total    int64
	}
	r := result.(la)
	return r.auctions, r.total, nil
}

func (s *AuctionService) InvalidateCache(ctx context.Context, auctionID string) {
	key := fmt.Sprintf("auction:%s", auctionID)
	s.redis.Del(ctx, key)
}

func (s *AuctionService) cacheAuction(ctx context.Context, auction e.Auction) {
	data, err := json.Marshal(auction)
	if err != nil {
		return
	}
	key := fmt.Sprintf("auction:%s", auction.AuctionID)
	s.redis.Set(ctx, key, data, s.cacheTTL)
}

func (s *AuctionService) getCachedAuction(ctx context.Context, auctionID string) *e.Auction {
	key := fmt.Sprintf("auction:%s", auctionID)
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}
	var auction e.Auction
	if err := json.Unmarshal(data, &auction); err != nil {
		return nil
	}
	return &auction
}
