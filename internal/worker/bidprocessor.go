package worker

import (
	"context"
	"errors"
	"time"

	e "auction-platform/internal/entity"
	kafkaclient "auction-platform/internal/infrastruct/kafka"
	kd "auction-platform/internal/infrastruct/kafka/dto"
	"auction-platform/internal/metrics"
	"auction-platform/internal/repo"
	re "auction-platform/internal/repo/errors"
	"auction-platform/internal/service"

	log "github.com/sirupsen/logrus"
)

type BidProcessor struct {
	bidService  service.Bids
	auctionRepo repo.Auctions
	bidRepo     repo.Bids
	producer    *kafkaclient.Producer
	metrics     *metrics.Metrics
	endTopic    string
}

func NewBidProcessor(
	bServ service.Bids,
	aRepo repo.Auctions,
	bRepo repo.Bids,
	producer *kafkaclient.Producer,
	m *metrics.Metrics,
	endTopic string,
) *BidProcessor {
	return &BidProcessor{
		bidService:  bServ,
		auctionRepo: aRepo,
		bidRepo:     bRepo,
		producer:    producer,
		metrics:     m,
		endTopic:    endTopic,
	}
}

func (p *BidProcessor) StartExpiryChecker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.Info("Auction expiry checker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.checkExpired(ctx)
		}
	}
}

func (p *BidProcessor) checkExpired(ctx context.Context) {
	expired, err := p.auctionRepo.GetExpired(ctx)
	if err != nil {
		log.Errorf("Failed to get expired auctions: %v", err)
		return
	}

	for _, auction := range expired {
		p.finishAuction(ctx, auction)
	}
}

func (p *BidProcessor) finishAuction(ctx context.Context, auction e.Auction) {
	highestBid, err := p.bidService.GetHighestBid(ctx, auction.AuctionID)
	winnerID := ""
	finalPrice := auction.StartPrice

	if err == nil {
		winnerID = highestBid.BidderID
		finalPrice = highestBid.Amount
	} else if !errors.Is(err, re.ErrNotFound) {
		log.Errorf("Failed to get highest bid for auction %s: %v", auction.AuctionID, err)
		return
	}

	if err := p.auctionRepo.FinishAuction(ctx, auction.AuctionID, winnerID, finalPrice); err != nil {
		log.Errorf("Failed to finish auction %s: %v", auction.AuctionID, err)
		return
	}

	totalBids, _ := p.bidService.CountByAuction(ctx, auction.AuctionID)

	event := kd.AuctionEndedEvent{
		AuctionID:  auction.AuctionID,
		WinnerID:   winnerID,
		FinalPrice: finalPrice,
		TotalBids:  totalBids,
	}
	p.producer.Publish(ctx, p.endTopic, auction.AuctionID, event)

	p.metrics.AuctionsFinished.Inc()
	p.metrics.ActiveAuctions.Dec()

	log.Infof("Auction finished [%s] winner=%s price=%.2f bids=%d",
		auction.AuctionID, winnerID, finalPrice, totalBids)
}
