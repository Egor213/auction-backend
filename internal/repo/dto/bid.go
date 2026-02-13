package repodto

import e "auction-platform/internal/entity"

type CreateBidInput struct {
	BidID     string
	AuctionID string
	BidderID  string
	Amount    float64
	Status    e.BidStatus
}
