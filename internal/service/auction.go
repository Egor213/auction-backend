package service

import (
	"context"
	"errors"

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

	log "github.com/sirupsen/logrus"
)

type AuctionService struct {
	auctionRepo repo.Auctions
	breaker     *circuitbreaker.CircuitBreaker
	retryer     *retry.Retryer
	metrics     *metrics.Metrics
}

func NewAuctionService(
	aRepo repo.Auctions,
	breaker *circuitbreaker.CircuitBreaker,
	retryer *retry.Retryer,
	m *metrics.Metrics,
) *AuctionService {
	return &AuctionService{
		auctionRepo: aRepo,
		breaker:     breaker,
		retryer:     retryer,
		metrics:     m,
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
	s.metrics.AuctionsCreated.Inc()
	s.metrics.ActiveAuctions.Inc()

	return auction, nil
}

func (s *AuctionService) GetAuction(ctx context.Context, auctionID string) (e.Auction, error) {
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
