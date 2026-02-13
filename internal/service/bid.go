package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	e "auction-platform/internal/entity"
	"auction-platform/internal/infrastruct/circuitbreaker"
	kafkaclient "auction-platform/internal/infrastruct/kafka"
	kd "auction-platform/internal/infrastruct/kafka/dto"
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

type BidService struct {
	auctionRepo repo.Auctions
	bidRepo     repo.Bids
	producer    *kafkaclient.Producer
	redis       *redis.Client
	breaker     *circuitbreaker.CircuitBreaker
	retryer     *retry.Retryer
	metrics     *metrics.Metrics
	bidTopic    string
	resultTopic string
}

func NewBidService(
	aRepo repo.Auctions,
	bRepo repo.Bids,
	producer *kafkaclient.Producer,
	rdb *redis.Client,
	breaker *circuitbreaker.CircuitBreaker,
	retryer *retry.Retryer,
	m *metrics.Metrics,
	bidTopic string,
	resultTopic string,
) *BidService {
	return &BidService{
		auctionRepo: aRepo,
		bidRepo:     bRepo,
		producer:    producer,
		redis:       rdb,
		breaker:     breaker,
		retryer:     retryer,
		metrics:     m,
		bidTopic:    bidTopic,
		resultTopic: resultTopic,
	}
}

func (s *BidService) PlaceBid(ctx context.Context, in sd.PlaceBidInput) (e.Bid, error) {
	repoIn := smap.ToCreateBidRepoInput(in)

	result, cbErr := s.breaker.Execute("postgres", func() (any, error) {
		var bid e.Bid
		err := s.retryer.Do(ctx, "create_bid", func() error {
			var e error
			bid, e = s.bidRepo.Create(ctx, repoIn)
			return e
		})
		return bid, err
	})
	if cbErr != nil {
		log.Error(errutils.WrapPathErr(cbErr))
		return e.Bid{}, se.ErrCannotCreateBid
	}

	bid := result.(e.Bid)

	event := kd.BidPlacedEvent{
		BidID:     bid.BidID,
		AuctionID: bid.AuctionID,
		BidderID:  bid.BidderID,
		Amount:    bid.Amount,
		Timestamp: bid.CreatedAt,
	}

	s.metrics.BidsPlaced.Inc()
	s.metrics.BidAmountHistogram.Observe(bid.Amount)

	if err := s.producer.Publish(ctx, s.bidTopic, bid.AuctionID, event); err != nil {
		log.Error(errutils.WrapPathErr(err))
		if processErr := s.ProcessBidEvent(ctx, event); processErr != nil {
			log.Error(errutils.WrapPathErr(processErr))
		}
		return e.Bid{}, se.ErrCannotPublishEvent
	}

	return bid, nil
}

func (s *BidService) ProcessBidEvent(ctx context.Context, event kd.BidPlacedEvent) error {
	lockKey := fmt.Sprintf("lock:auction:%s", event.AuctionID)
	lockVal, err := s.acquireLock(ctx, lockKey, 5*time.Second)
	if err != nil {
		return errutils.WrapPathErr(err)
	}
	defer s.releaseLock(ctx, lockKey, lockVal)

	auction, err := s.auctionRepo.GetByID(ctx, event.AuctionID)
	if err != nil {
		s.rejectBid(ctx, event, se.ErrNotFoundAuction.Error())
		return se.ErrNotFoundAuction
	}

	if auction.Status != e.AuctionStatusActive {
		s.rejectBid(ctx, event, se.ErrAuctionEnded.Error())
		return se.ErrAuctionEnded
	}

	if event.BidderID == auction.SellerID {
		s.rejectBid(ctx, event, se.ErrSellerCannotBid.Error())
		return se.ErrSellerCannotBid
	}

	if event.Amount < auction.CurrentBid+auction.MinStep {
		s.rejectBid(ctx, event, fmt.Sprintf("bid must be >= %.2f", auction.CurrentBid+auction.MinStep))
		return se.ErrBidTooLow
	}

	if err := s.auctionRepo.UpdateCurrentBid(ctx, event.AuctionID, event.Amount); err != nil {
		log.Error(errutils.WrapPathErr(err))
		return se.ErrCannotUpdateBid
	}

	if err := s.bidRepo.UpdateStatus(ctx, event.BidID, e.BidStatusAccepted); err != nil {
		log.Error(errutils.WrapPathErr(err))
		return se.ErrCannotUpdateBid
	}

	cacheKey := fmt.Sprintf("auction:%s", event.AuctionID)
	s.redis.Del(ctx, cacheKey)

	s.publishResult(ctx, event, string(e.BidStatusAccepted), "")
	s.metrics.BidsAccepted.Inc()

	return nil
}

func (s *BidService) rejectBid(ctx context.Context, event kd.BidPlacedEvent, reason string) {
	s.bidRepo.UpdateStatus(ctx, event.BidID, e.BidStatusRejected)
	s.publishResult(ctx, event, string(e.BidStatusRejected), reason)
	s.metrics.BidsRejected.Inc()
	log.Infof("Bid rejected [%s]: %s", event.BidID, reason)
}

func (s *BidService) publishResult(ctx context.Context, event kd.BidPlacedEvent, status, reason string) {
	result := kd.BidResultEvent{
		BidID:     event.BidID,
		AuctionID: event.AuctionID,
		BidderID:  event.BidderID,
		Amount:    event.Amount,
		Status:    status,
		Reason:    reason,
	}
	s.producer.Publish(ctx, s.resultTopic, event.AuctionID, result)
}

func (s *BidService) GetBidsByAuction(ctx context.Context, auctionID string, limit int) ([]e.Bid, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	result, cbErr := s.breaker.Execute("postgres", func() (any, error) {
		var bids []e.Bid
		err := s.retryer.Do(ctx, "create_bid", func() error {
			var e error
			bids, e = s.bidRepo.ListByAuction(ctx, auctionID, limit)
			return e
		})
		return bids, err
	})

	if cbErr != nil {
		log.Error(errutils.WrapPathErr(cbErr))
		return nil, se.ErrCannotGetBids
	}

	bids, _ := result.([]e.Bid)
	return bids, nil
}

func (s *BidService) acquireLock(ctx context.Context, key string, ttl time.Duration) (string, error) {
	val := fmt.Sprintf("%d", time.Now().UnixNano())
	ok, err := s.redis.SetNX(ctx, key, val, ttl).Result()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("lock already held")
	}
	return val, nil
}

func (s *BidService) releaseLock(ctx context.Context, key, value string) {
	var unlockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then 
		return redis.call("del", KEYS[1]) 
	else 
		return 0 
	end
	`)
	unlockScript.Run(ctx, s.redis, []string{key}, value)
}

func (s *BidService) GetHighestBid(ctx context.Context, auctionID string) (e.Bid, error) {
	bid, err := s.bidRepo.GetHighestByAuction(ctx, auctionID)
	if err != nil {
		if errors.Is(err, re.ErrNotFound) {
			return e.Bid{}, se.ErrNotFoundBid
		}
		return e.Bid{}, err
	}
	return bid, nil
}

func (s *BidService) CountByAuction(ctx context.Context, auctionID string) (int, error) {
	return s.bidRepo.CountByAuction(ctx, auctionID)
}
